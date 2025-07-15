package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/MuhamedUsman/letshare/internal/bgtask"
	"github.com/MuhamedUsman/letshare/internal/config"
	"github.com/MuhamedUsman/letshare/internal/domain"
	"github.com/MuhamedUsman/letshare/internal/network"
	"github.com/MuhamedUsman/letshare/internal/webui"
	"hash/crc32"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultPort  = 80
	TestHTTPPort = 8080
)

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

func (t tlog) relayActiveDown(n int, force bool) {
	if force {
		t.activeDownCh <- n
		return
	}
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
	ActiveDowns   int
	alreadyLogged map[string]struct{}
	// Option to let others on the same LAN to stopHandler this instance from hosting
	Stoppable bool
}

func New(stoppable bool, logCh chan<- Log, activeDownCh chan<- int) *Server {
	ctx, cancel := context.WithCancel(bgtask.Get().ShutdownCtx())
	l := tlog{logCh: logCh, activeDownCh: activeDownCh}
	return &Server{
		FilePaths:     make(map[uint32]string),
		log:           l,
		mu:            new(sync.Mutex),
		StopCtx:       ctx,
		StopCtxCancel: cancel,
		alreadyLogged: make(map[string]struct{}),
		Stoppable:     stoppable,
	}
}

// StartServer starts an HTTP/HTTPS server that serves files from Server.FilePaths.
// It binds to the machine's outbound IP address and handles graceful shutdown.
// NOTE: This must run first before MDNS entry is published as it dynamically determines
// the port to bind to, based on tls certificate availability.
// For more info see GetPort() && Server.configureServer().
//
// Returns:
//   - error: An error if the server fails to start, encounters issues during shutdown,
//     or if background tasks cannot be properly terminated.
//
// Note:
//   - Uses GetOutboundIP() to determine the IP address for binding.
//   - Will wait up to 2 seconds for server shutdown & 5 seconds for background tasks.
func (s *Server) StartServer(filePaths ...string) error {
	server, err := s.configureServer()
	if err != nil {
		return err
	}

	s.setFilePaths(filePaths...)
	defer func() {
		s.deleteTempFiles()
		close(s.log.logCh)
		close(s.log.activeDownCh)
	}()

	s.log.info("Starting server", "Addr", server.Addr)
	errChan := s.listenAndShutdown(server)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err // caller has context
	}
	s.log.info("Shutting down server", "Addr", server.Addr)
	if err = <-errChan; err != nil {
		return fmt.Errorf("server shutting down: %w", err)
	}
	return nil
}

// configureServer sets up the HTTP server with the necessary configurations.
//
// Returns:
//   - *http.Server: Configured HTTP server instance.
//   - error: An error if there is an issue during process.
func (s *Server) configureServer() (*http.Server, error) {
	addr, err := network.GetOutboundIP()
	if err != nil {
		return nil, err
	}

	var proto http.Protocols
	proto.SetUnencryptedHTTP2(true)
	proto.SetHTTP1(true)

	server := &http.Server{
		Addr:              fmt.Sprint(addr, ":", GetPort()),
		Handler:           s.routes(),
		ReadTimeout:       4 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       10 * time.Second,
		Protocols:         &proto,
	}
	return server, nil
}

func (s *Server) ShutdownServer() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.StopCtxCancel()
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
		// if the app is shutting down, we need to shut down fast, so shutdownCtx as parent
		ctx, cancel := context.WithTimeout(bgtask.Get().ShutdownCtx(), 2*time.Second)
		defer func() {
			cancel()
			close(errChan)
		}()
		if err := server.Shutdown(ctx); err != nil {
			errChan <- err
		}
	}()
	return errChan
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	base := newChain(s.recoverPanic, s.disallowOSHostnames, s.secureHeaders)

	fileServer := http.FileServer(http.FS(webui.Files))
	mux.Handle("GET /static/", base.then(fileServer))
	mux.Handle("GET /{$}", base.thenFunc(s.indexFilesHandler))
	mux.Handle("GET /{id}", base.thenFunc(s.serveFileHandler))
	mux.Handle("POST /stop", base.thenFunc(s.stopHandler))
	return mux
}

type indexFileTemplateData struct {
	HostUsername string
	TotalFiles   int
	TotalSize    int64
	Files        []*domain.FileInfo
}

// indexFilesHandler creates an HTTP handler that serves file indexes for Server.FilePaths.
// it returns a JSON-formatted directory listings.
// If an error occurs while reading the directory or generating the JSON response,
// an error response will be returned using serverErrorResponse.
func (s *Server) indexFilesHandler(w http.ResponseWriter, r *http.Request) {
	var fsInfos []*domain.FileInfo
	for k, v := range s.FilePaths {
		stat, err := os.Lstat(v)
		if err != nil {
			s.serverErrorResponse(w, r, err)
			return
		}
		fsInfo := &domain.FileInfo{
			AccessID: k,
			Name:     stat.Name(),
			Size:     stat.Size(),
		}
		fsInfos = append(fsInfos, fsInfo)
	}
	sortByNameAsc(fsInfos)

	logReq := shouldLogReq(r.RemoteAddr)
	reqBy := r.Header.Get("X-Requested-By")
	if reqBy == "" {
		reqBy = strings.Split(r.RemoteAddr, ":")[0]
	}

	if r.Header.Get("Accept") == "application/json" {
		if err := s.writeJSON(w, envelop{"fileIndexes": fsInfos}, http.StatusOK, nil); err != nil {
			s.serverErrorResponse(w, r, err)
		}
	} else {
		data := indexFileTemplateData{
			HostUsername: getConfig().Personal.Username,
			TotalFiles:   len(fsInfos),
			TotalSize:    getTotalFileSize(fsInfos),
			Files:        fsInfos,
		}
		if err := s.render(w, http.StatusOK, "fileIndexes.tmpl.html", data); err != nil {
			s.serverErrorResponse(w, r, err)
			return
		}
		if logReq {
			s.log.info("Your username was requested", "ReqBy", reqBy)
		}
	}

	if logReq {
		s.log.info("File indexes were requested", "ReqBy", reqBy)
	}
}

func (s *Server) serveFileHandler(w http.ResponseWriter, r *http.Request) {
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
	k := fmt.Sprint(accessID, ":", strings.Split(r.RemoteAddr, ":")[0])
	_, ok = s.alreadyLogged[k]
	shouldLog := shouldLogReq(r.RemoteAddr) && r.Method == http.MethodGet && !ok

	if shouldLog {
		s.alreadyLogged[k] = struct{}{} // mark as logged
		reqBy := r.Header.Get("X-Requested-By")
		if reqBy == "" {
			reqBy = strings.Split(r.RemoteAddr, ":")[0]
		}
		s.log.info("Serving file", "File", filename, "ReqBy", reqBy)
	}
	s.incActiveConn()       // this doesn't block
	defer s.decActiveConn() // this blocks

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
			msg := "Shutdown initiated, it may take maximum of 7 seconds to shutdown."
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

func (s *Server) incActiveConn() {
	s.mu.Lock()
	s.ActiveDowns++
	s.log.relayActiveDown(s.ActiveDowns, false)
	s.mu.Unlock()
}

func (s *Server) decActiveConn() {
	s.mu.Lock()
	s.ActiveDowns--
	s.log.relayActiveDown(s.ActiveDowns, true)
	s.mu.Unlock()
}

func getConfig() config.Config {
	cfg, err := config.Get()
	if err != nil && errors.Is(err, config.ErrNoConfig) {
		cfg, _ = config.Load()
	}
	return cfg
}

// not logging the request from the same machine
func shouldLogReq(addr string) bool {
	reqIp := strings.Split(addr, ":")[0]
	ip, _ := network.GetOutboundIP()
	return ip.String() != reqIp
}

func sortByNameAsc(fi []*domain.FileInfo) {
	slices.SortFunc(fi, func(a, b *domain.FileInfo) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
}

func getTotalFileSize(fi []*domain.FileInfo) int64 {
	var totalSize int64
	for _, f := range fi {
		totalSize += f.Size
	}
	return totalSize
}
