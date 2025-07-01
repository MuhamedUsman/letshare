package mdns

import (
	"context"
	"testing"
)

func TestMDNSDiscoveryAndPublication(t *testing.T) {
	m := Get()
	instance := "TestInstance"
	hostname := "TestHost.local"
	username := "TestUsername"
	port := uint16(8080)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Start publishing in background
	go func() {
		if err := m.Publish(ctx, instance, hostname, username, port); err != nil {
			t.Error(err)
			return
		}
	}()

	go func() {
		if err := m.Browse(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	<-m.NotifyOnChange()

	if entry, ok := m.Entries()[instance]; ok {
		if !(entry.Hostname == hostname && entry.Port == port && entry.Owner == username) {
			t.Errorf("Expected entry with Hostname: %s, Port: %d, Owner: %s, got Hostname: %s, Port: %d, Owner: %s",
				hostname, port, username, entry.Hostname, entry.Port, entry.Owner)
		}
	}
}
