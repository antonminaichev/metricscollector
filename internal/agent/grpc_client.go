package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"

	"github.com/antonminaichev/metricscollector/internal/crypto"
	metricsv1 "github.com/antonminaichev/metricscollector/internal/proto/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	_ "google.golang.org/grpc/encoding/gzip" // gzip for gRPC
)

func RunGRPCPublisher(
	ctx context.Context,
	addr string,
	hashKey string,
	cryptoKeyPath string,
	jobs <-chan Metrics,
	reportInterval int,
) error {
	log.Printf("[AGENT gRPC] dialing %s ...", addr)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[AGENT gRPC] dial failed: %v", err)
		return err
	}
	defer func() {
		_ = conn.Close()
		log.Printf("[AGENT gRPC] connection closed")
	}()
	log.Printf("[AGENT gRPC] connected")

	client := metricsv1.NewMetricsServiceClient(conn)
	log.Printf("[AGENT gRPC] opening stream Push()")
	stream, err := client.Push(ctx)
	if err != nil {
		log.Printf("[AGENT gRPC] open stream failed: %v", err)
		return err
	}
	log.Printf("[AGENT gRPC] stream opened")

	defer func() {
		log.Printf("[AGENT gRPC] closing stream (waiting for summary)")
		res, cerr := stream.CloseAndRecv()
		if cerr != nil {
			log.Printf("[AGENT gRPC] close&recv failed: %v", cerr)
			return
		}
		log.Printf("[AGENT gRPC] summary accepted=%d failed=%d", res.GetAccepted(), res.GetFailed())
	}()

	// Ключ шифрования (опционально)
	var pubKey *rsa.PublicKey
	if cryptoKeyPath != "" {
		if k, lerr := crypto.LoadPublicKey(cryptoKeyPath); lerr == nil {
			pubKey = k
			log.Printf("[AGENT gRPC] public key loaded: %s", cryptoKeyPath)
		} else {
			log.Printf("[AGENT gRPC] WARNING: failed to load public key (%s): %v", cryptoKeyPath, lerr)
		}
	} else {
		log.Printf("[AGENT gRPC] no public key configured (sending plaintext payloads)")
	}

	agentIP := getLocalIP()
	log.Printf("[AGENT gRPC] agent_ip=%s report_interval=%ds", agentIP, reportInterval)

	ticker := time.NewTicker(time.Duration(reportInterval) * time.Second)
	defer ticker.Stop()

	const maxBatch = 5000
	batch := make([]Metrics, 0, 512)

	flush := func(reason string) {
		if len(batch) == 0 {
			return
		}
		start := time.Now()
		var sent int
		for _, m := range batch {
			// gzip(JSON)
			rawJSON, _ := json.Marshal(m)
			var gz bytes.Buffer
			gzw, _ := gzip.NewWriterLevel(&gz, gzip.BestSpeed)
			if _, err := gzw.Write(rawJSON); err != nil {
				log.Printf("[AGENT gRPC] gzip write failed: %v", err)
				continue
			}
			if err := gzw.Close(); err != nil {
				log.Printf("[AGENT gRPC] gzip close failed: %v", err)
				continue
			}

			payload := gz.Bytes()
			encrypted := false

			// Шифрование (если есть ключ)
			if pubKey != nil {
				if enc, err := crypto.EncryptRSA(pubKey, payload); err == nil {
					payload = enc
					encrypted = true
				} else {
					log.Printf("[AGENT gRPC] encrypt failed (fallback to plaintext): %v", err)
				}
			}

			// HMAC от фактических байтов payload
			var hashHex string
			if hashKey != "" {
				mac := hmac.New(sha256.New, []byte(hashKey))
				mac.Write(payload)
				hashHex = hex.EncodeToString(mac.Sum(nil))
			}

			req := &metricsv1.PushRequest{
				Payload:   payload,
				Hash:      hashHex,
				AgentIp:   agentIP,
				Encrypted: encrypted,
			}
			if err := stream.Send(req); err != nil {
				log.Printf("[AGENT gRPC] send failed: %v", err)
				continue
			}
			sent++
		}
		log.Printf("[AGENT gRPC] flushed %d/%d metrics in %s (%s)",
			sent, len(batch), time.Since(start).Truncate(time.Millisecond), reason)
		// очистка батча(flush)
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush("ctx done")
			return nil

		case <-ticker.C:
			flush("tick")

		case m, ok := <-jobs:
			if !ok {
				flush("jobs closed")
				return nil
			}
			batch = append(batch, m)
			if len(batch) >= maxBatch {
				flush("maxBatch")
			}
		}
	}
}
