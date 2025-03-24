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
	if err := run(2, 10, "http://localhost:8080"); err != nil {
		panic(err)
	}
}

func run(pollinterval int, reportInterval int, host string) error {
	client := &http.Client{}

	// Создаем канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем функции параллельно
	go agent.CollectMetrics(pollinterval)
	go agent.PostMetric(client, reportInterval, host)

	// Ждем сигнала завершения
	<-sigChan
	log.Println("Получен сигнал завершения работы")
	return nil
}
