package mdns

import (
	"context"
	"errors"
	"fmt"
	"github.com/brutella/dnssd"
	"github.com/brutella/dnssd/log"
	"sync"
)

const (
	DefaultInstance = "letshare"
	mdnsService     = "_http._tcp"
)

var (
	defaultOwnerKey = "defaultOwner"
	once            = new(sync.Once)
	mdns            *MDNS
)

// ServiceEntry represents a discovered mDNS service with its network details.
type ServiceEntry struct {
	Hostname, IP string
	Port         int
}

// ServiceEntries represents discovered mDNS services where:
// - Key: Service instance name
// - Value: Service network details (Hostname, IP, Port)
type ServiceEntries map[string]ServiceEntry

// MDNS handles multicast DNS service registration and discovery on local networks.
// It provides methods to publish services and discover other services on the network.
type MDNS struct {
	mu sync.RWMutex
	// Stores discovered mDNS entries
	entries ServiceEntries
	// name of the user currently occupying the default instance
	defaultOwner string
	// Channel to signal updates or changes
	notifyCh chan struct{}
}

// Get creates and returns a new MDNS instance.
// The returned instance is initialized with an empty service entries map.
//
// This function uses a singleton pattern to ensure only one MDNS instance
// exists across the application. Subsequent calls return the same instance.
func Get() *MDNS {
	once.Do(func() {
		mdns = &MDNS{entries: make(ServiceEntries)}
		log.Info.Disable() // Disable logging for dnssd package
	})
	return mdns
}

// Publish advertises a service via multicast DNS on available network interfaces.
// The service uses the "_http._tcp" service type and "local." domain.
// The service remains advertised until the provided context is canceled.
//
// Parameters:
//   - ctx: Context that controls the service advertisement lifetime
//   - instance: The service instance name (visible to other devices)
//   - Hostname: The Hostname of the device providing the service
//   - Port: The Port number on which the service is available
//
// Returns an error if service registration fails.
func (r *MDNS) Publish(ctx context.Context, instance, host string, username string, port int) error {
	cfg := dnssd.Config{
		Name: instance,
		Type: mdnsService,
		Host: host,
		Port: port,
		Text: map[string]string{defaultOwnerKey: username},
	}
	sv, err := dnssd.NewService(cfg)
	if err != nil {
		return fmt.Errorf("registering mdns entry: %v", err)
	}

	rp, err := dnssd.NewResponder()
	if err != nil {
		return fmt.Errorf("creating mdns responder: %v", err)
	}

	hdl, err := rp.Add(sv)
	if err != nil {
		return fmt.Errorf("adding service to mdns responder: %v", err)
	}

	go func() {
		<-ctx.Done()
		rp.Remove(hdl)
	}()

	// if it's the default instance, store the owner
	if instance == DefaultInstance {
		r.mu.Lock()
		r.defaultOwner = instance
		r.mu.Unlock()
	}

	if err = rp.Respond(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("responding to mdns requests: %v", err)
	}
	return nil
}

// Discover continuously discovers mDNS services on the local network.
// Discovered services are automatically stored in the internal entries map
// and can be retrieved using the Entries() method.
//
// Parameters:
//   - ctx: Context that controls the discovery process lifetime
//   - service: The service type to discover (e.g., "_http._tcp.local.")
//
// Returns an error if the discovery process fails to start.
func (r *MDNS) Discover(ctx context.Context) error {
	addFunc := dnssd.AddFunc(func(e dnssd.BrowseEntry) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.entries[e.Name] = ServiceEntry{
			Hostname: e.Host,
			IP:       e.IPs[0].To4().String(), // Assuming the first IP is the primary one
			Port:     e.Port,
		}

		// If this is the default instance, update the owner
		if e.Name == DefaultInstance {
			r.defaultOwner = e.Text[defaultOwnerKey]
		}

		if r.notifyCh != nil {
			close(r.notifyCh)
		}
	})
	rmvFunc := dnssd.RmvFunc(func(e dnssd.BrowseEntry) {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.entries, e.Name)

		// If the default instance is removed, clear the owner
		if e.Name == DefaultInstance {
			r.defaultOwner = ""
		}

		if r.notifyCh != nil {
			close(r.notifyCh)
		}
	})
	service := fmt.Sprintf("%s.local.", mdnsService)
	return dnssd.LookupType(ctx, service, addFunc, rmvFunc)
}

// NotifyOnChange blocks until a change occurs in the discovered mDNS entries.
func (r *MDNS) NotifyOnChange() {
	r.notifyCh = make(chan struct{})
	<-r.notifyCh
}

// Entries returns a copy of all currently discovered mDNS services.
// This method is safe for concurrent use from multiple goroutines.
//
// Returns:
//   - ServiceEntries: A map of service instance names to their network details
func (r *MDNS) Entries() ServiceEntries {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries
}

// DefaultOwner returns the username of the owner of the default instance.
func (r *MDNS) DefaultOwner() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultOwner
}
