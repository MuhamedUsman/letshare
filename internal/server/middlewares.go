package server

import (
	"fmt"
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
				s.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) disallowOsHostnames(next http.Handler) http.Handler {
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
