package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/utility"
	"github.com/grandcat/zeroconf"
	"github.com/justinas/alice"
	"io"
	"log"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	// Option to let others on the same LAN to stop this instance from hosting
	Stoppable bool
	// Once Done, the server will exit
	StopCtx context.Context
	// Cancel func for StopCtx
	StopCancel context.CancelFunc
	// Every Goroutine must run through BT Run function
	BT *utility.BackgroundTask
	mu *sync.Mutex
	// indicates if the server is idling or currently serving files
	ActiveDowns int
}

func New() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		BT:         utility.NewBackgroundTask(),
		StopCtx:    ctx,
		StopCancel: cancel,
		mu:         new(sync.Mutex),
		Stoppable:  true,
	}
}

type CopyStat struct {
	// current number of file being copied
	N int // 0: Err occurred before copying
	// error encountered while copying the file
	Err error
}

type CopyStatChan <-chan CopyStat

// CopyFilesToDir copies the specified files to the target directory.
// It returns a CopyStatChan that reports progress and errors during the operation.
//
// The returned channel receives a CopyStat after each file operation, containing:
// - N: The number of files processed so far (1-indexed)
// - Err: Any error that occurred during the operation
//
// The channel will be closed when either:
// - All files have been copied successfully
// - An error occurs (copying stops at first error)
// - The Server's shutdown context is canceled
//
// If the shutdown context is canceled during copying, all copied files
// will be deleted from the target directory.
func (s *Server) CopyFilesToDir(dir string, files ...string) CopyStatChan {
	ch := make(chan CopyStat)
	s.BT.Run(func(shutdownCtx context.Context) {
		defer close(ch)
		for i, f := range files {
			select {
			case <-shutdownCtx.Done():
				// Once canceled during copying delete all the files that are copied
				for _, file := range files {
					copiedFilepath := filepath.Join(dir, filepath.Base(file))
					_ = os.Remove(copiedFilepath) // ignore any error
				}
				return
			default:
				n := i + 1
				if _, err := os.Stat(dir); err != nil {
					if os.IsNotExist(err) {
						ch <- CopyStat{N: n, Err: fmt.Errorf("file does not exist: %v", f)}
					} else {
						ch <- CopyStat{N: n, Err: fmt.Errorf("cannot access file %q: %v", f, err.Error())}
					}
					return
				}
				destPath := filepath.Join(dir, filepath.Base(f))
				if err := copyFile(f, destPath); err != nil {
					ch <- CopyStat{N: n, Err: fmt.Errorf("copying file %q to %q: %v", f, destPath, err.Error())}
					return
				}
				ch <- CopyStat{N: n, Err: nil}
			}
		}
	})
	return ch
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file %s: %v", src, err)
	}
	defer func() {
		if err = s.Close(); err != nil {
			slog.Error("failed to close source file", "file", src, "Err", err)
		}
	}()
	d, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file %s: %v", dst, err)
	}
	defer func() {
		if err = d.Close(); err != nil {
			slog.Error("failed to close destination file", "file", dst, "Err", err)
		}
	}()
	if _, err = io.Copy(d, s); err != nil {
		return fmt.Errorf("copying file %s to %s: %v", src, dst, err)
	}
	return nil
}

// StartServerForDir starts an HTTP server that serves files from the specified directory.
// It binds to the machine's outbound IP address on port 80 and handles graceful shutdown
// on context cancellation or OS termination signals (SIGINT, SIGTERM).
//
// The function sets up proper timeouts for read operations and idle connections.
// The server routes are configured through the Server.routes method which should
// handle serving files from the provided directory.
//
// Parameters:
//   - dir: The directory path to serve files from. Must be a valid directory.
//
// Returns:
//   - error: An error if the server fails to start, encounters issues during shutdown,
//     or if background tasks cannot be properly terminated.
//
// Panics:
//   - If the provided 'dir' path exists but is not a directory.
//
// Note:
//   - Uses GetOutboundIP() to determine the IP address for binding.
//   - Will wait up to 5 seconds for server shutdown & 5 seconds for background tasks.
func (s *Server) StartServerForDir(dir string) error {
	if info, _ := os.Stat(dir); info != nil && !info.IsDir() {
		panic(fmt.Sprintf("%q is not a directory", dir))
	}
	ipAddr, err := GetOutboundIP()
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              fmt.Sprint(ipAddr.String(), ":80"),
		ReadTimeout:       4 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       10 * time.Second,
		Handler:           s.routes(dir),
	}
	errChan := s.listenAndShutdown(server)
	slog.Info("Starting Server", "address", server.Addr)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server listning on address %q", server.Addr)
	}
	if err = <-errChan; err != nil {
		return fmt.Errorf("server shutting down: %v", err)
	}
	if err = s.BT.Shutdown(5 * time.Second); err != nil {
		return fmt.Errorf("shutting down background tasks: %v", err)
	}
	return nil
}

func (s *Server) listenAndShutdown(server *http.Server) chan error {
	errChan := make(chan error)
	go func() {
		defer close(errChan)
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-s.StopCtx.Done():
		case <-quit:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			errChan <- fmt.Errorf("shutting down server: %v", err)
		}
	}()
	return errChan
}

func (s *Server) routes(dir string) http.Handler {
	mux := http.NewServeMux()
	panicRecover := alice.New(s.recoverPanic)
	mux.Handle("GET /", panicRecover.Then(s.JsonFileServer(dir)))
	mux.Handle("GET /stop", panicRecover.ThenFunc(s.Stop))
	return mux
}

type FSInfo struct {
	Name    string
	Path    string
	Size    int64
	Type    string // MIME type, if not resolved then extension
	ModTime time.Time
}

// JsonFileServer creates an HTTP handler that serves files from the specified directory.
// For the root URL ("/"), it returns a JSON-formatted directory listing containing details
// of all files (not subdirectories) in the directory. For other paths, it serves the
// requested file directly.
//
// The JSON response for the root path includes an array of FSInfo objects with the following
// properties for each file:
//   - Name: The name of the file
//   - Path: URL-escaped path to access the file, prefixed with "/"
//   - Size: File size in bytes
//   - Type: MIME type (determined by file extension) or extension name if MIME type is unknown
//   - ModTime: Last modification time of the file
//
// If an error occurs while reading the directory or generating the JSON response,
// an error response will be returned using serverErrorResponse.
func (s *Server) JsonFileServer(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.ActiveDowns++
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			s.ActiveDowns--
			s.mu.Unlock()
		}()
		entries, err := os.ReadDir(dir)
		if r.URL.Path != "/" { // that means user is accessing some file
			http.ServeFile(w, r, path.Join(dir, path.Clean(r.URL.Path)))
			return
		}
		var fsInfos []FSInfo
		for _, entry := range entries {
			// only host files
			if entry.IsDir() {
				continue
			}
			var finfo os.FileInfo
			finfo, err = entry.Info()
			ext := path.Ext(entry.Name())
			fileType := mime.TypeByExtension(ext)
			if fileType == "" {
				fileType = strings.TrimPrefix(ext, ".")
			}
			fsInfo := FSInfo{
				Name:    entry.Name(),
				Path:    path.Join("/", url.PathEscape(entry.Name())),
				Size:    finfo.Size(),
				Type:    fileType,
				ModTime: finfo.ModTime(),
			}
			fsInfos = append(fsInfos, fsInfo)
		}
		if err = s.writeJSON(w, envelop{"Shared": fsInfos}, http.StatusOK, nil); err != nil {
			s.serverErrorResponse(w, r, err)
			return
		}
	})
}

// Stop handles HTTP requests to stop the server.
// Only works when the server is stoppable and not actively serving files.
//
// Returns:
// - Success (202 Accepted): When shutdown is initiated
// - Error: When the server is not stoppable or is currently serving files
func (s *Server) Stop(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	c := s.ActiveDowns
	s.mu.Unlock()
	if s.Stoppable && c == 0 {
		s.StopCancel()
		msg := "Shutdown initiated, it may take maximum of 10 seconds to shutdown the server."
		if err := s.writeJSON(w, envelop{"status": msg}, http.StatusAccepted, nil); err != nil {
			s.serverErrorResponse(w, r, err)
		}
		return
	}
	if !s.Stoppable {
		s.notStoppableResponse(w, r)
		return
	}
	s.notIdleResponse(w, r)
}

// GetOutboundIP gets the preferred outbound ip of this machine
func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, fmt.Errorf("dialing to get outbound ip address: %v", err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

// PublishEntry publishes a Multicast DNS entry over available network interfaces.
// It uses the predefined service "_http._tcp", domain "local.", and port 80.
// The function blocks until the provided context is canceled, at which point
// it shuts down the mDNS server and returns.
//
// Parameters:
//   - ctx: Context that controls the lifetime of the mDNS service
//   - instance: The instance name to publish (visible as the service name)
//   - info: Optional text records to associate with the service
//
// Returns an error if the service registration fails.
func PublishEntry(ctx context.Context, instance string, info ...string) error {
	server, err := zeroconf.Register(instance, "_http._tcp", "local.", 80, info, nil)
	if err != nil {
		return fmt.Errorf("registering zeroconf: %v", err)
	}
	defer server.Shutdown()
	<-ctx.Done()
	return nil
}

func (s *Server) ResolveEntry() {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)
	s.BT.Run(func(shutdownCtx context.Context) {
		for {
			select {
			case <-shutdownCtx.Done():
				return
			case entry, ok := <-entries:
				if !ok {
					log.Println("No more entries")
					return
				}

				log.Println("Received entry:", entry)
			}
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = resolver.Browse(ctx, "_http._tcp", "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}
	<-ctx.Done()
}
