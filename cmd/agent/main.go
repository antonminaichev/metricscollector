package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/antonminaichev/metricscollector/internal/agent"
)

func main() {
	if err := parseFlags(); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		return
	}
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	client := &http.Client{}

	// Создаем канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем функции параллельно
	go agent.CollectMetrics(flagPollInterval)
	go agent.PostMetric(client, flagReportInterval, flagRunAddr)

	// Ждем сигнала завершения
	<-sigChan
	log.Println("Получен сигнал завершения работы")
	return nil
}
