package main

import (
	"flag"
	"fmt"
)

var flagRunAddr string

func parseFlags() error {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "{Host:port} for server")
	flag.Parse()
	if len(flag.Args()) > 0 {
		return fmt.Errorf("unknown flags: %v", flag.Args())
	}

	return nil
}
