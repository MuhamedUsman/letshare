package mdns

import (
	"context"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/network"
	"github.com/betamos/zeroconf"
	"maps"
	"net/netip"
	"sync"
)

const (
	// UsernameKey used to store the username of the service owner in mDNS TXT records
	UsernameKey     = "username"
	DefaultInstance = "letshare"
	Domain          = "local"
	mdnsService     = "_http._tcp"
)

var (
	typ  = zeroconf.NewType(mdnsService)
	once sync.Once
	mdns *MDNS
)

// ServiceEntry represents a discovered mDNS service with its network details.
type ServiceEntry struct {
	Owner, Hostname, IP string
	Port                uint16
}

// ServiceEntries represents discovered mDNS services where:
// - Key: Service instance name
// - Value: Service network details (Hostname, IP, Port)
type ServiceEntries map[string]ServiceEntry

// MDNS handles multicast DNS service registration and discovery on local networks.
// It provides methods to publish services and discover other services on the network.
type MDNS struct {
	// Mutex to protect access to entries & notifyCh
	mu sync.RWMutex
	// Stores discovered mDNS entries
	entries ServiceEntries
	// channel to broadcast changes to entries
	notifyCh chan struct{}
	pub, bro *zeroconf.Client
}

// Get creates and returns a new MDNS instance.
// The returned instance is initialized with an empty service entries map.
//
// This function uses a singleton pattern to ensure only one MDNS instance
// exists across the application. Subsequent calls return the same instance.
func Get() *MDNS {
	once.Do(func() {
		mdns = &MDNS{
			entries:  make(ServiceEntries),
			notifyCh: make(chan struct{}),
		}
	})
	return mdns
}

// Publish registers a service instance with mDNS.
//
// Params:
//   - ctx: Context to control the lifetime of the service registration.
//   - instance: Unique name for the service instance. (e.g., "letshare")
//   - hostname: Hostname of the machine hosting the service. (e.g., "my-computer.local")
//   - username: Username of the service owner, used in TXT records. (e.g., "john_doe")
//   - port: Port on which the service is running. (e.g., 80)
//
// Returns:
//   - error: An error if the service registration fails, otherwise nil.
func (r *MDNS) Publish(ctx context.Context, instance, hostname, username string, port uint16) error {
	s := zeroconf.NewService(typ, instance, port)
	s.Hostname = hostname

	usr := fmt.Sprintf("%s=%s", UsernameKey, username)
	s.Text = []string{usr}

	addr, err := network.GetOutboundIP()
	if err != nil {
		return fmt.Errorf("getting outbound IP address: %w", err)
	}
	s.Addrs = []netip.Addr{addr}

	r.pub, err = zeroconf.New().Publish(s).Open()
	if err != nil {
		return fmt.Errorf("publishing mDNS service: %w", err)
	}

	defer func() {
		_ = r.pub.Close()
	}()

	<-ctx.Done()
	return err
}

// Browse discovers mDNS services on the local network.
// It listens for changes in the mDNS service entries and updates the internal state.
// The function blocks until the provided context is done.
// It uses a callback function to handle service events (added, updated, removed).
func (r *MDNS) Browse(ctx context.Context) error {
	bf := func(e zeroconf.Event) {
		r.mu.Lock()
		switch e.Op {
		case zeroconf.OpAdded, zeroconf.OpUpdated:
			se := ServiceEntry{
				Owner:    extractOwner(e.Text),
				Hostname: e.Hostname,
				IP:       e.Addrs[0].String(),
				Port:     e.Port,
			}
			r.entries[e.Name] = se
			close(r.notifyCh)                // Notify the change
			r.notifyCh = make(chan struct{}) // Reset the channel for next notification

		case zeroconf.OpRemoved:
			if _, ok := r.entries[e.Name]; ok {
				delete(r.entries, e.Name)
				close(r.notifyCh)
				r.notifyCh = make(chan struct{})
			}
		}
		r.mu.Unlock()
	}

	var err error
	r.bro, err = zeroconf.New().Browse(bf, typ).Open()
	if err != nil {
		return fmt.Errorf("browsing mDNS services: %w", err)
	}

	defer func() {
		_ = r.bro.Close()
	}()

	<-ctx.Done()
	return err
}

// NotifyOnChange returns a channel that blocks until a change occurs in the discovered mDNS entries.
// the channel is closed when a change happens, and it is reset for the next notification.
func (r *MDNS) NotifyOnChange() <-chan struct{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.notifyCh
}

// Entries returns a copy of all currently discovered mDNS services.
// This method is safe for concurrent use from multiple goroutines.
//
// Returns:
//   - ServiceEntries: A map of service instance names to their network details(ServiceEntry).
func (r *MDNS) Entries() ServiceEntries {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dst := make(ServiceEntries, len(r.entries))
	maps.Copy(dst, r.entries)
	return dst
}

func (r *MDNS) ReloadPublisher() {
	r.pub.Reload()
}

func (r *MDNS) ReloadBrowser() {
	r.bro.Reload()
}

func extractOwner(s []string) string {
	kl := len(UsernameKey)
	for _, kv := range s {
		if kv == "" {
			continue
		}
		kvl := len(kv)
		if kvl > kl && kv[:kl] == UsernameKey && kv[kl] == '=' {
			return kv[kl+1:]
		}
	}
	return ""
}
