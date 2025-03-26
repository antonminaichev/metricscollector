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
	flag.StringVar(&flagRunAddr, "addr", "localhost:8080", "{Host:port} for server")
	flag.IntVar(&flagReportInterval, "report", 10, "report interval")
	flag.IntVar(&flagPollInterval, "poll", 2, "poll interval")
	flag.Parse()
	if len(flag.Args()) > 0 {
		return fmt.Errorf("unknown flags: %v", flag.Args())
	}

	return nil
}
