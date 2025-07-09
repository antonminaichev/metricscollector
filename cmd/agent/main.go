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

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	client := &http.Client{}
	cfg, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	jobs := make(chan agent.Metrics, cfg.RateLimit*3)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agent.CollectMetrics(ctx, cfg.PollInterval, jobs)
	go agent.CollectSystemMetrics(ctx, cfg.PollInterval, jobs)

	var wg sync.WaitGroup
	wg.Add(cfg.RateLimit - 1)
	for i := 0; i < cfg.RateLimit; i++ {
		go func() {
			defer wg.Done()
			agent.MetricWorker(client, cfg.Address, cfg.HashKey, jobs, cfg.ReportInterval)
		}()
	}

	// Ждем сигнала завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Получен сигнал завершения работы")

	close(jobs)
	wg.Wait()
	cancel()
	log.Println("agent stopped")
	return nil
}
