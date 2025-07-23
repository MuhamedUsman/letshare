package server

import (
	"net/http"
	"os"
	"slices"
	"strings"
)

type chain []func(http.Handler) http.Handler

func newChain(h ...func(http.Handler) http.Handler) chain {
	return h
}

func (c chain) thenFunc(h http.HandlerFunc) http.Handler {
	return c.then(h)
}

func (c chain) then(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c) {
		h = mw(h)
	}
	return h
}

func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				s.serverErrorResponse(w, r)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) disallowOSHostnames(next http.Handler) http.Handler {
	osHostname, err := os.Hostname()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := strings.Replace(r.Host, ".local", "", 1) // remove .local suffix if present
		if err == nil && strings.EqualFold(h, osHostname) {
			s.noOSHostnameAllowedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; font-src 'self'")
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}
