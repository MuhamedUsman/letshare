package mdns

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestMDNSDiscoveryAndPublication(t *testing.T) {
	m := Get()
	instance := "TestInstance"
	host := "TestHost"
	username := "TestUsername"

	slog.SetLogLoggerLevel(slog.LevelWarn)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Start publishing in background
	go func() {
		if err := m.Publish(ctx, instance, host, username, 80); err != nil {
			t.Error(err)
			return
		}
	}()

	go func() {
		if err := m.Discover(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	// Retry loop instead of fixed delay
	var found bool
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)
		if e, ok := m.Entries()[instance]; ok && e.Hostname == host {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("Failed to discover the published instance within timeout")
	}
}
