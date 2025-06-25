package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"github.com/MuhamedUsman/letshare/internal/config"
	"github.com/MuhamedUsman/letshare/internal/domain"
	"github.com/MuhamedUsman/letshare/internal/mdns"
	"github.com/MuhamedUsman/letshare/internal/network"
	"github.com/justinas/alice"
	"hash/crc32"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrNonIdle = errors.New("server is not idle (serving files)")

type Log struct {
	Msg  string
	Args []any
}

// tlog is used to lazy Log messages from the server which tui will display
type tlog struct {
	logCh        chan<- Log
	activeDownCh chan<- int
}

func (t tlog) info(msg string, args ...any) {
	select {
	case t.logCh <- Log{Msg: msg, Args: args}:
	default:
	}
}

func (t tlog) relayActiveDown(n int) {
	select {
	case t.activeDownCh <- n:
	default:
	}
}

type Server struct {
	// file paths to be served, [K: accessID, V: filepath]
	FilePaths map[uint32]string
	log       tlog
	mu        *sync.Mutex
	// Once Done, the server will exit
	StopCtx context.Context
	// Cancel func for StopCtx
	StopCtxCancel context.CancelFunc
	// notifyCh is used to notify for the server shutdown request when ActiveDowns > 0
	notifyCh chan<- string // X-Requested-By header value
	// indicates if the server is idling or currently serving files
	ActiveDowns int
	// Option to let others on the same LAN to stopHandler this instance from hosting
	Stoppable bool
}

func New(stoppable bool, logCh chan<- Log, activeDownCh chan<- int) *Server {
	ctx, cancel := context.WithCancel(bgtask.Get().ShutdownCtx())
	log := tlog{logCh: logCh, activeDownCh: activeDownCh}
	return &Server{
		FilePaths:     make(map[uint32]string),
		log:           log,
		mu:            new(sync.Mutex),
		StopCtx:       ctx,
		StopCtxCancel: cancel,
		Stoppable:     stoppable,
	}
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
func (s *Server) StartServer(filePaths ...string) error {
	ipAddr, err := network.GetOutboundIP()
	if err != nil {
		return err
	}
	addr := fmt.Sprint(ipAddr.To4(), ":80")
	if config.TestFlag {
		addr = fmt.Sprint(ipAddr.To4(), ":8080")
	}

	server := &http.Server{
		Addr:              addr,
		ReadTimeout:       4 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       10 * time.Second,
		Handler:           s.routes(),
	}

	s.setFilePaths(filePaths...)
	defer func() {
		s.deleteTempFiles()
		close(s.log.logCh)
	}()

	errChan := s.listenAndShutdown(server)
	s.log.info("Starting server", "Addr", addr)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err // caller has context
	}
	s.log.info("Shutting down server", "Addr", addr)
	if err = <-errChan; err != nil {
		return fmt.Errorf("server shutting down: %w", err)
	}
	return nil
}

func (s *Server) ShutdownServer(force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ActiveDowns == 0 || force {
		s.StopCtxCancel()
		return nil
	}
	return ErrNonIdle
}

func (s *Server) NotifyForShutdownReqWhenNotIdle(ch chan<- string) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	username := getConfig().Personal.Username
	if err := s.writeJSON(w, envelop{mdns.UsernameKey: username}, http.StatusOK, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
	if shouldLogReq(r.RemoteAddr) {
		reqBy := r.Header.Get("X-Requested-By")
		if reqBy == "" {
			reqBy = strings.Split(r.RemoteAddr, ":")[0]
		}
		s.log.info("Your username was requested", "ReqBy", reqBy)
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
		return
	}
	if shouldLogReq(r.RemoteAddr) {
		reqBy := r.Header.Get("X-Requested-By")
		if reqBy == "" {
			reqBy = strings.Split(r.RemoteAddr, ":")[0]
		}
		s.log.info("File indexes were requested", "ReqBy", reqBy)
	}
}

func (s *Server) serveFileHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.ActiveDowns++
	s.log.relayActiveDown(s.ActiveDowns)
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.ActiveDowns--
		s.log.relayActiveDown(s.ActiveDowns)
		s.mu.Unlock()
	}()

	accessID := r.PathValue("id")
	id, err := strconv.ParseUint(accessID, 10, 32)
	if err != nil { // Invalid access ID
		s.notFoundResponse(w, r)
		return
	}

	filePath, ok := s.FilePaths[uint32(id)]
	if !ok {
		s.notFoundResponse(w, r)
		return
	}
	filename := filepath.Base(filePath)

	var reqBy string
	shouldLog := shouldLogReq(r.RemoteAddr) && r.Method == "GET" && r.Header.Get("Range") == ""

	if shouldLog {
		reqBy = r.Header.Get("X-Requested-By")
		if reqBy == "" {
			reqBy = strings.Split(r.RemoteAddr, ":")[0]
		}
		s.log.info("Serving file", "File", filename, "ReqBy", reqBy)
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	http.ServeFile(w, r, filePath)

	if shouldLog {
		s.log.info("Serving completed", "File", filename, "ReqBy", reqBy)
	}
}

// stopHandler handles HTTP requests to shut down the server.
// Only works when the server is stoppable and not actively serving files.
//
// Returns:
// - Success (202 Accepted): When shutdown is initiated
// - Error: When the server is not stoppable or is currently serving files
func (s *Server) stopHandler(w http.ResponseWriter, r *http.Request) {
	reqBy := r.Header.Get("X-Requested-By")
	if reqBy == "" {
		reqBy = strings.Split(r.RemoteAddr, ":")[0]
	}

	if shouldLogReq(r.RemoteAddr) {
		s.log.info("Server shutdown request", "ReqBy", reqBy)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.ActiveDowns

	if s.Stoppable {
		if c == 0 {
			s.StopCtxCancel()
			msg := "Shutdown initiated, it may take maximum of 10 seconds."
			s.log.info(msg)
			if err := s.writeJSON(w, envelop{"status": msg}, http.StatusAccepted, nil); err != nil {
				s.serverErrorResponse(w, r, err)
			}
			return
		}

		if s.notifyCh != nil {
			s.notifyCh <- reqBy
			close(s.notifyCh)
			s.notifyCh = nil
		}
		s.notIdleResponse(w, r)
		return
	}
	s.notStoppableResponse(w, r)
}

// setFilePaths sets the file paths to be served by the server.
func (s *Server) setFilePaths(filePaths ...string) {
	for _, p := range filePaths {
		hash := crc32.ChecksumIEEE([]byte(p))
		s.FilePaths[hash] = p
	}
}

func (s *Server) deleteTempFiles() {
	s.log.info("Deleting temporary files")
	for _, p := range s.FilePaths {
		if strings.HasPrefix(p, os.TempDir()) {
			if err := os.Remove(p); err != nil {
				slog.Error("deleting temp files", "err", err)
			}
		}
	}
}

func getConfig() config.Config {
	cfg, err := config.Get()
	if err != nil && errors.Is(err, config.ErrNoConfig) {
		cfg, _ = config.Load()
	}
	return cfg
}

func shouldLogReq(addr string) bool {
	reqIp := strings.Split(addr, ":")[0]
	ip, _ := network.GetOutboundIP()
	return ip.String() != reqIp
}
