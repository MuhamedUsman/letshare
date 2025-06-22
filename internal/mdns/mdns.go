package mdns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/cfg"
	"github.com/MuhamedUsman/letshare/internal/network"
	"github.com/brutella/dnssd"
	"github.com/brutella/dnssd/log"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	DefaultInstance = "letshare"
	// UsernameKey used to store the username of the service owner in mDNS TXT records
	UsernameKey = "username"
	mdnsService = "_http._tcp"
	domain      = "local."
)

var (
	ErrNotFound      = errors.New("mdns instance not found")
	ErrInstanceOwned = errors.New("mdns instance already owned")

	once sync.Once
	mdns *MDNS
)

func init() {
	log.Info.Disable()
}

// ServiceEntry represents a discovered mDNS service with its network details.
type ServiceEntry struct {
	Owner, Hostname, IP string
	Port                int
}

// ServiceEntries represents discovered mDNS services where:
// - Key: Service instance name
// - Value: Service network details (Hostname, IP, Port)
type ServiceEntries map[string]ServiceEntry

// MDNS handles multicast DNS service registration and discovery on local networks.
// It provides methods to publish services and discover other services on the network.
type MDNS struct {
	// Channel to signal updates or changes
	notifyCh chan struct{}
	// Mutex to protect access to entries and defaultOwner
	mu sync.RWMutex
	// Stores discovered mDNS entries
	entries ServiceEntries
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
//   - host: The Hostname of the device providing the service
//   - username: username of the instance's Owner
//   - Port: The Port number on which the service is available
//
// Returns an error if service registration fails.
func (r *MDNS) Publish(ctx context.Context, instance, host, username string, port int) error {

	if cfg.TestFlag {
		port = 8080
	}

	ip, err := network.GetOutboundIP()
	if err != nil {
		return err
	}

	cfg := dnssd.Config{
		Name: instance,
		Type: mdnsService,
		Host: host,
		Port: port,
		IPs:  []net.IP{ip},
		Text: map[string]string{UsernameKey: username},
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
		if err := r.triggerRefresh(); err != nil {
			slog.Error("Error refreshing mdns service", "err", err)
		}
	}()

	if err = rp.Respond(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("responding to mdns requests: %v", err)
	}
	return nil
}

func (r *MDNS) triggerRefresh() error {
	addFunc := dnssd.AddFunc(func(e dnssd.BrowseEntry) {})
	rvmFunc := dnssd.RmvFunc(func(e dnssd.BrowseEntry) {})
	service := fmt.Sprintf("%s.%s", mdnsService, domain)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := dnssd.LookupType(ctx, service, addFunc, rvmFunc); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("triggring refresh: %v", err)
	}
	return nil
}

// Discover continuously discovers mDNS services on the local network.
// Discovered services are automatically stored in the internal entries map
// and can be retrieved using the Entries() method.
//
// Parameters:
//   - ctx: Context that controls the discovery process lifetime
//
// Returns an error if the discovery process fails to start.
func (r *MDNS) Discover(ctx context.Context) error {
	addFunc := dnssd.AddFunc(func(e dnssd.BrowseEntry) {
		owner := e.Text[UsernameKey]

		if owner == "" { // get the username by an http call
			addr := fmt.Sprintf("%s:%d", e.IPs[0].To4().String(), e.Port)
			var err error
			owner, err = getOwner(addr)
			if err != nil {
				slog.Error("error getting owner name", "err", err)
			}
		}

		r.mu.Lock()
		defer r.mu.Unlock()

		r.entries[e.Name] = ServiceEntry{
			Owner:    owner,
			Hostname: e.Host,
			IP:       e.IPs[0].To4().String(), // Assuming the first IP is the primary one
			Port:     e.Port,
		}

		if r.notifyCh != nil {
			close(r.notifyCh)
		}

	})
	rmvFunc := dnssd.RmvFunc(func(e dnssd.BrowseEntry) {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.entries, e.Name)
		if r.notifyCh != nil {
			close(r.notifyCh)
		}

	})
	service := fmt.Sprintf("%s.%s", mdnsService, domain)
	return dnssd.LookupType(ctx, service, addFunc, rmvFunc)
}

// NotifyOnChange blocks until a change occurs in the discovered mDNS entries.
func (r *MDNS) NotifyOnChange() {
	r.notifyCh = make(chan struct{})
	<-r.notifyCh
	r.notifyCh = nil
}

// Entries returns a copy of all currently discovered mDNS services.
// This method is safe for concurrent use from multiple goroutines.
//
// Returns:
//   - ServiceEntries: A map of service instance names to their network details(ServiceEntry).
func (r *MDNS) Entries() ServiceEntries {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries
}

func getOwner(addr string) (string, error) {
	c := http.Client{Timeout: 2 * time.Second}
	addr = fmt.Sprintf("http://%s/owner", addr)
	resp, err := c.Get(addr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %v", err)
	}

	var ownerName struct {
		Username string `json:"username"`
	}
	if err = json.Unmarshal(b, &ownerName); err != nil {
		return "", fmt.Errorf("parsing response: %v", err)
	}
	return ownerName.Username, nil
}
