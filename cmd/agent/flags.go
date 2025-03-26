package main

import (
	"flag"
	"fmt"
)

var (
	flagRunAddr        string
	flagReportInterval int
	flagPollInterval   int
)

func parseFlags() error {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "{Host:port} for server")
	flag.IntVar(&flagReportInterval, "r", 10, "report interval")
	flag.IntVar(&flagPollInterval, "p", 2, "poll interval")
	flag.Parse()
	if len(flag.Args()) > 0 {
		return fmt.Errorf("unknown flags: %v", flag.Args())
	}

	return nil
}
