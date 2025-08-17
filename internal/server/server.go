package server

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/antonminaichev/metricscollector/internal/crypto"
	"github.com/antonminaichev/metricscollector/internal/logger"
	"github.com/antonminaichev/metricscollector/internal/server/middleware"
	"github.com/antonminaichev/metricscollector/internal/server/router"
	"github.com/antonminaichev/metricscollector/internal/server/storage"
	db "github.com/antonminaichev/metricscollector/internal/server/storage/database"
	fs "github.com/antonminaichev/metricscollector/internal/server/storage/file"
	ms "github.com/antonminaichev/metricscollector/internal/server/storage/memstorage"

	"go.uber.org/zap"
)

// Config stores server setting.
type Config struct {
	Address            string `env:"ADDRESS"`
	LogLevel           string `env:"LOG_LEVEL"`
	StoreInterval      int    `env:"STORE_INTERVAL"`
	FileStoragePath    string `env:"FILE_STORAGE_PATH"`
	Restore            bool   `env:"RESTORE"`
	DatabaseConnection string `env:"DATABASE_DSN"`
	HashKey            string `env:"KEY"`
	CryptoKey          string `env:"CRYPTO_KEY"`
	TrustedSubnet      string `env:"TRUSTED_SUBNET"`
}

func StartServer(addr string, storage storage.Storage, hashKey string, privKeyPath string, trustedCIDR string) error {
	privKey, err := crypto.LoadPrivateKey(privKeyPath)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	server := &http.Server{
		Addr: addr,
		Handler: logger.WithLogging(
			middleware.HashHandler(
				middleware.RSADecryptMiddleware(privKey)(
					middleware.GzipHandler(
						router.NewRouter(storage, trustedCIDR),
					),
				),
				hashKey,
			),
		),
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			errCh <- err
			return
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logger.Log.Info("Shutdown signal received, stopping HTTP…")

		shCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if err := server.Shutdown(shCtx); err != nil {
			logger.Log.Warn("Shutdown error", zap.Error(err))
		}

		logger.Log.Info("Server shutdown complete")
		return nil

	case err := <-errCh:
		return err
	}
}

func SetupStorage(DSN string, fspath string, restore bool, storeInterval int) (storage.Storage, error) {
	if DSN != "" {
		logger.Log.Info("Connecting to database", zap.String("dsn", DSN))
		return db.NewPostgresStorage(DSN)
	}

	if fspath != "" {
		fs, err := fs.NewFileStorage(fspath, logger.Log)
		if err != nil {
			return nil, err
		}
		logger.Log.Info("Using file storage", zap.String("path", fspath))

		if restore {
			if err := fs.LoadMetrics(); err != nil {
				logger.Log.Warn("Failed to restore metrics from file", zap.Error(err))
			}
		}

		go startPeriodicSave(fs, storeInterval)
		return fs, nil
	}

	logger.Log.Info("Using in-memory storage")
	return ms.NewMemoryStorage(), nil
}

func startPeriodicSave(fs *fs.FileStorage, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := fs.SaveMetrics(); err != nil {
			logger.Log.Error("Failed to save metrics to file", zap.Error(err))
		}
	}
}
