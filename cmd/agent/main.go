package main

import (
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

	go agent.CollectMetrics(cfg.PollInterval, jobs)
	go agent.CollectSystemMetrics(cfg.PollInterval, jobs)

	var wg sync.WaitGroup
	for i := 0; i < cfg.RateLimit; i++ {
		wg.Add(1)
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
	log.Println("agent stopped")
	return nil
}
