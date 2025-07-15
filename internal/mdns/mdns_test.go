package mdns

import (
	"context"
	"github.com/stretchr/testify/assert"
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
		err := m.Publish(ctx, instance, hostname, username, port)
		assert.NoError(t, err, "Failed to publish mDNS service")
	}()
	go func() {
		err := m.Browse(ctx)
		assert.NoError(t, err, "Failed to browse mDNS services")
	}()

	<-m.NotifyOnChange()

	entry, ok := m.Entries()[instance]
	assert.True(t, ok, "Expected service entry to be present after publishing")
	assert.Equal(t, entry.Hostname, hostname, "Expected hostname to match")
	assert.Equal(t, entry.Port, port, "Expected port to match")
	assert.Equal(t, entry.Owner, username, "Expected owner to match")
}
