package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigDefaults(t *testing.T) {
	os.Clearenv()
	os.Args = []string{"cmd"}

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, "localhost:8080", cfg.Address)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 2, cfg.ReportInterval)
	assert.Equal(t, 30, cfg.RateLimit)
}
