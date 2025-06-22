package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"github.com/MuhamedUsman/letshare/internal/cfg"
	"github.com/MuhamedUsman/letshare/internal/client"
	"github.com/MuhamedUsman/letshare/internal/domain"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/network"
	"github.com/justinas/alice"
	"log/slog"
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
	// notifyCh is used to notify for the server shutdown request when ActiveDowns > 0
	notifyCh chan string // X-Requested-By header value
	// Option to let others on the same LAN to stopHandler this instance from hosting
	Stoppable bool
}

func New() *Server {
	ctx, cancel := context.WithCancel(bgtask.Get().ShutdownCtx())
	return &Server{
		FilePaths:     make(map[string]string),
		StopCtx:       ctx,
		StopCtxCancel: cancel,
		mu:            new(sync.Mutex),
		ActiveDowns:   1,
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
// The server routes are configured through the Server.routes method which should
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
	ipAddr, err := network.GetOutboundIP()
	if err != nil {
		return err
	}
	addr := fmt.Sprint(ipAddr.To4(), ":80")
	if cfg.TestFlag {
		addr = fmt.Sprint(ipAddr.To4(), ":8080")
	}
	server := &http.Server{
		Addr:              addr,
		ReadTimeout:       4 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       10 * time.Second,
		Handler:           s.routes(),
	}

	errChan := s.listenAndShutdown(server)
	slog.Info("Starting Server", "address", server.Addr)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err // caller has context
	}
	if err = <-errChan; err != nil {
		return fmt.Errorf("server shutting down: %w", err)
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

func (s *Server) NotifyForShutdownReqWhenNotIdle(ch chan string) {
	s.notifyCh = ch
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

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	panicRecover := alice.New(s.recoverPanic)
	mux.Handle("GET /{$}", panicRecover.ThenFunc(s.indexFilesHandler))
	mux.Handle("GET /{id}", panicRecover.ThenFunc(s.serveFileHandler))
	mux.Handle("GET /owner", panicRecover.ThenFunc(s.ownerNameHandler))
	mux.Handle("POST /stop", panicRecover.ThenFunc(s.stopHandler))
	return mux
}

func (s *Server) ownerNameHandler(w http.ResponseWriter, r *http.Request) {
	config, err := client.LoadConfig()
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
	username := config.Personal.Username
	if err = s.writeJSON(w, envelop{mdns.UsernameKey: username}, http.StatusOK, nil); err != nil {
		s.serverErrorResponse(w, r, err)
	}
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

// stopHandler handles HTTP requests to shut down the server.
// Only works when the server is stoppable and not actively serving files.
//
// Returns:
// - Success (202 Accepted): When shutdown is initiated
// - Error: When the server is not stoppable or is currently serving files
func (s *Server) stopHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	c := s.ActiveDowns
	s.mu.Unlock()
	if s.Stoppable {
		if c == 0 {
			s.StopCtxCancel()
			msg := "Shutdown initiated, it may take maximum of 10 seconds to shutdown the server."
			if err := s.writeJSON(w, envelop{"status": msg}, http.StatusAccepted, nil); err != nil {
				s.serverErrorResponse(w, r, err)
			}
			return
		}
		s.mu.Lock()
		if s.notifyCh != nil {
			s.notifyCh <- r.Header.Get("X-Requested-By")
			close(s.notifyCh)
			s.notifyCh = nil
		}
		s.mu.Unlock()
		s.notIdleResponse(w, r)
		return
	}
	s.notStoppableResponse(w, r)
}
