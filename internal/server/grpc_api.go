package server

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"strings"

	"github.com/antonminaichev/metricscollector/internal/crypto"
	"github.com/antonminaichev/metricscollector/internal/logger"
	metricsv1 "github.com/antonminaichev/metricscollector/internal/proto/metrics/v1"
	"github.com/antonminaichev/metricscollector/internal/server/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	_ "google.golang.org/grpc/encoding/gzip"
)

type grpcServer struct {
	metricsv1.UnimplementedMetricsServiceServer
	storage     storage.Storage
	hashKey     string
	privateKey  *rsa.PrivateKey
	trustedCIDR string
}

func StartGRPCServer(addr string, s storage.Storage, hashKey, privateKeyPath, trustedCIDR string) error {
	var priv *rsa.PrivateKey
	if privateKeyPath != "" {
		if k, err := crypto.LoadPrivateKey(privateKeyPath); err == nil {
			priv = k
			logger.Log.Info("grpc: private key loaded", zap.String("path", privateKeyPath))
		} else {
			logger.Log.Warn("grpc: failed to load private key", zap.Error(err), zap.String("path", privateKeyPath))
		}
	} else {
		logger.Log.Info("grpc: no private key configured (expect plaintext payloads)")
	}

	gs := grpc.NewServer()
	metricsv1.RegisterMetricsServiceServer(gs, &grpcServer{
		storage:     s,
		hashKey:     hashKey,
		privateKey:  priv,
		trustedCIDR: trustedCIDR,
	})

	logger.Log.Info("Starting gRPC server", zap.String("address", addr), zap.String("trusted_cidr", trustedCIDR))
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return gs.Serve(l)
}

func (g *grpcServer) Push(stream metricsv1.MetricsService_PushServer) error {
	var accepted, failed int64

	// Лог старта стрима и peer-адреса
	if p, ok := peer.FromContext(stream.Context()); ok && p.Addr != nil {
		logger.Log.Info("grpc stream opened", zap.String("peer", p.Addr.String()))
	} else {
		logger.Log.Info("grpc stream opened", zap.String("peer", "<unknown>"))
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			logger.Log.Info("grpc stream closing", zap.Int64("accepted", accepted), zap.Int64("failed", failed))
			return stream.SendAndClose(&metricsv1.PushResult{
				Accepted: accepted,
				Failed:   failed,
			})
		}
		if err != nil {
			logger.Log.Warn("grpc recv failed", zap.Error(err))
			return err
		}

		agentIP := strings.TrimSpace(req.GetAgentIp())
		if agentIP == "" {
			if p, ok := peer.FromContext(stream.Context()); ok && p.Addr != nil {
				host, _, _ := net.SplitHostPort(p.Addr.String())
				agentIP = host
			}
		}

		logger.Log.Info("grpc recv",
			zap.String("agent_ip", agentIP),
			zap.Bool("encrypted", req.GetEncrypted()),
			zap.Int("payload_len", len(req.GetPayload())),
		)

		// CIDR-проверка (если CIDR пуст — разрешаем)
		if !grpcIPAllowed(agentIP, g.trustedCIDR) {
			failed++
			logger.Log.Warn("grpc cidr reject",
				zap.String("ip", agentIP),
				zap.String("cidr", g.trustedCIDR),
			)
			continue
		}

		// HMAC
		if g.hashKey != "" && req.GetHash() != "" {
			mac := hmac.New(sha256.New, []byte(g.hashKey))
			mac.Write(req.GetPayload())
			exp := hex.EncodeToString(mac.Sum(nil))
			got := strings.ToLower(req.GetHash())
			if !hmac.Equal([]byte(got), []byte(strings.ToLower(exp))) {
				failed++
				logger.Log.Warn("grpc bad hmac", zap.String("expected", exp), zap.String("got", got))
				continue
			}
		}

		// Дешифрование (если требуется)
		data := req.GetPayload()
		if req.GetEncrypted() && g.privateKey != nil {
			dec, err := crypto.DecryptRSA(g.privateKey, data)
			if err != nil {
				failed++
				logger.Log.Warn("grpc decrypt failed", zap.Error(err))
				continue
			}
			data = dec
		}

		// Разgzip
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			failed++
			logger.Log.Warn("grpc gzip reader failed", zap.Error(err))
			continue
		}
		plain, err := io.ReadAll(zr)
		_ = zr.Close()
		if err != nil {
			failed++
			logger.Log.Warn("grpc gzip read failed", zap.Error(err))
			continue
		}

		// JSON -> storage.Metric
		var metric storage.Metric
		if err := json.Unmarshal(plain, &metric); err != nil {
			failed++
			logger.Log.Warn("grpc json unmarshal failed", zap.Error(err))
			continue
		}
		logger.Log.Info("grpc metric parsed",
			zap.String("id", metric.ID),
			zap.String("type", string(metric.MType)),
		)

		switch metric.MType {
		case storage.Counter:
			if metric.Delta == nil {
				failed++
				logger.Log.Warn("grpc metric delta=nil for counter", zap.String("id", metric.ID))
				continue
			}
			if err := g.storage.UpdateMetric(stream.Context(), metric.ID, storage.Counter, metric.Delta, nil); err != nil {
				failed++
				logger.Log.Warn("grpc update counter failed", zap.Error(err), zap.String("id", metric.ID))
				continue
			}
		case storage.Gauge:
			if metric.Value == nil {
				failed++
				logger.Log.Warn("grpc metric value=nil for gauge", zap.String("id", metric.ID))
				continue
			}
			if err := g.storage.UpdateMetric(stream.Context(), metric.ID, storage.Gauge, nil, metric.Value); err != nil {
				failed++
				logger.Log.Warn("grpc update gauge failed", zap.Error(err), zap.String("id", metric.ID))
				continue
			}
		default:
			failed++
			logger.Log.Warn("grpc unknown metric type", zap.String("id", metric.ID), zap.String("type", string(metric.MType)))
			continue
		}

		accepted++
		logger.Log.Info("grpc metric stored", zap.String("id", metric.ID), zap.String("type", string(metric.MType)))
	}
}

// Пустой trustedCIDR.
func grpcIPAllowed(ipStr, trustedCIDR string) bool {
	if strings.TrimSpace(trustedCIDR) == "" {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	_, n, err := net.ParseCIDR(trustedCIDR)
	if err != nil {
		return false
	}
	return n.Contains(ip)
}
