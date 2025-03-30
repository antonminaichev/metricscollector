package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

var (
	flagRunAddr        string
	flagReportInterval int
	flagPollInterval   int
)

func parseFlags() error {
	flag.StringVar(&flagRunAddr, "a", "http://localhost:8080", "{Host:port} for server")
	flag.IntVar(&flagReportInterval, "r", 10, "report interval")
	flag.IntVar(&flagPollInterval, "p", 2, "poll interval")
	flag.Parse()
	if len(flag.Args()) > 0 {
		return fmt.Errorf("unknown flags: %v", flag.Args())
	}
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}
	if envRepInt := os.Getenv("REPORT_INTERVAL"); envRepInt != "" {
		envRepInt, _ := strconv.Atoi(envRepInt)
		flagReportInterval = envRepInt
	}
	if envPollInt := os.Getenv("POLL_INTERVAL"); envPollInt != "" {
		envPollInt, _ := strconv.Atoi(envPollInt)
		flagReportInterval = envPollInt
	}
	return nil
}
