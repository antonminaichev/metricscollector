package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/antonminaichev/metricscollector/internal/agent"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func printBuildInfo() {
	v := buildVersion
	if v == "" {
		v = "N/A"
	}
	d := buildDate
	if d == "" {
		d = "N/A"
	}
	c := buildCommit
	if c == "" {
		c = "N/A"
	}

	log.Printf("Build version: %s\n", v)
	log.Printf("Build date: %s\n", d)
	log.Printf("Build commit: %s\n", c)
}

func run() error {
	printBuildInfo()

	cfg, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	jobs := make(chan agent.Metrics, cfg.RateLimit*3)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var collectWG sync.WaitGroup
	collectWG.Add(2)
	go func() {
		defer collectWG.Done()
		agent.CollectMetrics(ctx, cfg.PollInterval, jobs)
	}()
	go func() {
		defer collectWG.Done()
		agent.CollectSystemMetrics(ctx, cfg.PollInterval, jobs)
	}()

	var sendWG sync.WaitGroup

	if cfg.Mode == "grpc" {
		sendWG.Add(1)
		go func() {
			defer sendWG.Done()
			if err := agent.RunGRPCPublisher(ctx, cfg.GRPCAddress, cfg.HashKey, cfg.CryptoKey, jobs, cfg.ReportInterval); err != nil {
				log.Printf("grpc publisher error: %v", err)
			}
		}()
	} else {
		client := &http.Client{}
		sendWG.Add(cfg.RateLimit)
		for i := 0; i < cfg.RateLimit; i++ {
			go func() {
				defer sendWG.Done()
				agent.MetricWorker(client, cfg.Address, cfg.HashKey, jobs, cfg.ReportInterval, cfg.CryptoKey)
			}()
		}
	}

	// Ждем сигнала завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigChan
	log.Println("Получен сигнал завершения работы")
	cancel()

	collectWG.Wait()
	close(jobs)
	sendWG.Wait()

	log.Println("agent stopped")
	return nil
}
