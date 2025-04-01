package mdns

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMDNSDiscoveryAndPublication(t *testing.T) {
	mdns := New()
	instance := "TestInstance"
	var entries ServiceEntry

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	publishReady := make(chan struct{})
	// Start publishing in background
	go func() {
		close(publishReady) // signals publishing goroutine is in action
		err := mdns.Publish(ctx, instance)
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("publishing entry: %v", err)
		}
	}()

	<-publishReady
	// Lookup the published entry
	var err error
	entries, err = mdns.lookup(500 * time.Millisecond) // some timeout so the publishing may finish till then
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	// once the lookup finishes. canceling stops the Publishing
	cancel()

	// Verify results
	if hostname, ok := entries[instance]; !ok {
		t.Fatal("Failed to discover the published instance")
	} else if hostname == "" {
		t.Fatal("Discovered instance has empty hostname")
	}
}
