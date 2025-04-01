package mdns

import (
	"context"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/util"
	"github.com/grandcat/zeroconf"
	"strings"
	"sync"
	"time"
)

// ServiceEntry represents a map of discovered mDNS services where:
// - Key: Instance name of the service
// - Value: Host name of the device providing the service
type ServiceEntry map[string]string

// MDNS handles multicast DNS service registration and discovery on local networks.
// It provides methods to publish services and discover other services.
type MDNS struct {
	bt *util.BackgroundTask
	mu sync.RWMutex
	// Stores discovered mDNS entries
	entries ServiceEntry
}

// New creates and returns a new MDNS instance ready for use.
// The returned instance is initialized with an empty entries map
// and a background task manager.
func New() *MDNS {
	return &MDNS{
		bt:      util.NewBgTask(),
		entries: make(ServiceEntry),
	}
}

// Publish advertises a service via Multicast DNS over available network interfaces.
// It uses the predefined service "_http._tcp", domain "local.", and port 80.
// The service remains published until the provided context is canceled.
//
// Parameters:
//   - ctx: Context that controls the lifetime of the mDNS service
//   - instance: The instance name to publish (visible as the service name)
//   - info: Optional text records to associate with the service (key-value pairs)
//
// Returns an error if the service registration fails.
func (r *MDNS) Publish(ctx context.Context, instance string, info ...string) error {
	server, err := zeroconf.Register(instance, "_http._tcp", "local.", 80, info, nil)
	if err != nil {
		return fmt.Errorf("registering mdns entry through zeroconf: %v", err)
	}
	defer server.Shutdown()
	<-ctx.Done()
	return nil
}

// DiscoverMDNSEntries continuously discovers mDNS services on the local network
// at regular intervals. Discovered services are stored in the MDNS.entries field.
//
// Parameters:
//   - afterEach: Duration to wait between discovery attempts
//   - lookFor: Maximum duration to spend on each discovery attempt
//
// Returns a channel that will receive any errors encountered during discovery.
// The channel is closed when discovery stops.
//
// The discovery process continues until the Application exits.
// The entries map is cleared when discovery stops.
// The entries can be accessed through MDNS.Entries
func (r *MDNS) DiscoverMDNSEntries(afterEach, lookFor time.Duration) <-chan error {
	errCh := make(chan error)
	r.bt.Run(func(shutdownCtx context.Context) {
		defer func() {
			close(errCh)
			clear(r.entries)
		}()
		for {
			select {
			case <-shutdownCtx.Done():
				return
			case <-time.After(afterEach):
				entries, err := r.lookup(lookFor)
				if err != nil {
					errCh <- err
					return
				}
				r.mu.Lock()
				r.entries = entries // replace with updated entries
				r.mu.Unlock()
			}
		}
	})
	return errCh
}

// lookup performs a single discovery operation for "_http._tcp" services
// on the local network. It collects discovered services for the specified
// timeout duration.
//
// Parameters:
//   - timeout: Maximum duration to spend discovering services
//
// Returns:
//   - ServiceEntry: Map of discovered service instances to their hostnames
//   - error: An error if service discovery initialization fails
//
// This is an internal method used by DiscoverMDNSEntries.
func (r *MDNS) lookup(timeout time.Duration) (ServiceEntry, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("initializing mdns resolver: %v", err)
	}
	entriesCh := make(chan *zeroconf.ServiceEntry)
	entries := make(ServiceEntry)
	r.bt.Run(func(shutdownCtx context.Context) {
		for {
			select {
			case <-shutdownCtx.Done():
				return
			case e, ok := <-entriesCh:
				if !ok {
					return
				}
				r.mu.Lock()
				entries[e.Instance] = strings.TrimSuffix(e.HostName, ".") // K: Letshare | V: usman-v14.local
				r.mu.Unlock()
			}
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = resolver.Browse(ctx, "_http._tcp", "local.", entriesCh)
	if err != nil {
		return nil, fmt.Errorf("browsing mdns entries: %v", err)
	}
	<-ctx.Done()
	return entries, nil
}

// Entries returns a copy of the current set of discovered mDNS entries.
// It's safe to call this method concurrently from multiple goroutines.
//
// Returns:
//   - ServiceEntry: A map of service instance names to their host names
func (r *MDNS) Entries() ServiceEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries
}
