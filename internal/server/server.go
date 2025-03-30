package server

import (
	"context"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/common"
	"github.com/grandcat/zeroconf"
	"io"
	"log"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	BT *common.BackgroundTask
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

// SrvDir returns a http.handlerFunc serving the dir passed as parameter
func (s *Server) SrvDir(dir string) {
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
		Handler:           s.createSrvDirHandler(dir),
	}
	err = server.ListenAndServe()
	log.Fatal(err)
}

func (s *Server) createSrvDirHandler(dir string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", s.JsonFileServer(dir))
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
