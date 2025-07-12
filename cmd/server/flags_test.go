package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServerConfigDefaults(t *testing.T) {
	os.Clearenv()
	os.Args = []string{"cmd"}

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, "localhost:8080", cfg.Address)
	assert.Equal(t, "./metrics/metrics.json", cfg.FileStoragePath)
	assert.True(t, cfg.Restore)
}
