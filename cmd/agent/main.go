package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
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
	// Создаем канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем функции параллельно
	go agent.CollectMetrics(cfg.PollInterval)
	go agent.PostMetricsBatch(client, cfg.ReportInterval, cfg.Address, cfg.HashKey)

	// Ждем сигнала завершения
	<-sigChan
	log.Println("Получен сигнал завершения работы")
	return nil
}
