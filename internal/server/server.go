package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/domain"
	"github.com/justinas/alice"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrNonIdle = errors.New("server is not idle (serving files)")

type Server struct {
	// file paths to be served, [K: accessID, V: filepath]
	FilePaths map[string]string
	// Once Done, the server will exit
	StopCtx context.Context
	// Cancel func for StopCtx
	StopCtxCancel context.CancelFunc
	mu            *sync.Mutex
	// indicates if the server is idling or currently serving files
	ActiveDowns int
	// Option to let others on the same LAN to stop this instance from hosting
	Stoppable bool
}

func New() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		FilePaths:     make(map[string]string),
		StopCtx:       ctx,
		StopCtxCancel: cancel,
		mu:            new(sync.Mutex),
		ActiveDowns:   0,
		Stoppable:     true,
	}
}

// SetFilePaths sets the file paths to be served by the server.
func (s *Server) SetFilePaths(filePaths ...string) error {
	for _, p := range filePaths {
		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		id := hex.EncodeToString(randBytes)
		s.FilePaths[id] = p
	}
	return nil
}

// StartServer starts an HTTP server that serves files from Server.FilePaths.
// It binds to the machine's outbound IP address on port 2403 and handles graceful shutdown
// on context cancellation or OS termination signals (SIGINT, SIGTERM).
//
// The function sets up proper timeouts for read operations and idle connections.
// The server Routes are configured through the Server.Routes method which should
// handle serving files from the provided directory.
//
// Returns:
//   - error: An error if the server fails to start, encounters issues during shutdown,
//     or if background tasks cannot be properly terminated.
//
// Note:
//   - Uses GetOutboundIP() to determine the IP address for binding.
//   - Will wait up to 5 seconds for server shutdown & 5 seconds for background tasks.
func (s *Server) StartServer() error {
	ipAddr, err := GetOutboundIP()
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              fmt.Sprint(ipAddr.String(), ":80"),
		ReadTimeout:       4 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       10 * time.Second,
		Handler:           s.Routes(),
	}
	errChan := s.listenAndShutdown(server)
	slog.Info("Starting Server", "address", server.Addr)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server listning on address %q", server.Addr)
	}
	if err = <-errChan; err != nil {
		return fmt.Errorf("server shutting down: %v", err)
	}
	return nil
}

func (s *Server) ShutdownServer(force bool) error {
	if s.ActiveDowns == 0 || force {
		s.StopCtxCancel()
		return nil
	}
	return ErrNonIdle
}

func (s *Server) listenAndShutdown(server *http.Server) chan error {
	errChan := make(chan error)
	go func() {
		<-s.StopCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			cancel()
			close(errChan)
		}()
		if err := server.Shutdown(ctx); err != nil {
			errChan <- fmt.Errorf("shutting down server: %v", err)
		}
	}()
	return errChan
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	panicRecover := alice.New(s.recoverPanic)
	mux.Handle("GET /{$}", panicRecover.ThenFunc(s.indexFilesHandler))
	mux.Handle("GET /{id}", panicRecover.ThenFunc(s.serveFileHandler))
	mux.Handle("POST /stop", panicRecover.ThenFunc(s.Stop))
	return mux
}

// indexFilesHandler creates an HTTP handler that serves file indexes for Server.FilePaths.
// it returns a JSON-formatted directory listings.
// If an error occurs while reading the directory or generating the JSON response,
// an error response will be returned using serverErrorResponse.
func (s *Server) indexFilesHandler(w http.ResponseWriter, r *http.Request) {
	var fsInfos []domain.FileInfo
	for k, v := range s.FilePaths {
		stat, err := os.Lstat(v)
		if err != nil {
			s.serverErrorResponse(w, r, err)
			return
		}
		fsInfo := domain.FileInfo{
			AccessID: k,
			Name:     stat.Name(),
			Size:     stat.Size(),
			Type:     strings.TrimPrefix(filepath.Ext(stat.Name()), "."),
		}
		fsInfos = append(fsInfos, fsInfo)
	}
	if err := s.writeJSON(w, envelop{"fileIndexes": fsInfos}, http.StatusOK, nil); err != nil {
		s.serverErrorResponse(w, r, err)
	}
}

func (s *Server) serveFileHandler(w http.ResponseWriter, r *http.Request) {
	accessID := r.PathValue("id")
	filePath, ok := s.FilePaths[accessID]
	if !ok {
		s.notFoundResponse(w, r)
		return
	}
	filename := filepath.Base(filePath)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	http.ServeFile(w, r, filePath)
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
		s.StopCtxCancel()
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
