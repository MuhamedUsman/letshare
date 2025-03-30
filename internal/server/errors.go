package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func (s *Server) errorResponse(w http.ResponseWriter, _ *http.Request, status int, message any) {
	data := envelop{"errors": message}
	if err := s.writeJSON(w, data, status, nil); err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	s.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (s *Server) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error(err.Error())
	debug.PrintStack()
	message := "the Server encountered a problem and could not process your request"
	s.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (s *Server) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource cannot be found"
	s.errorResponse(w, r, http.StatusNotFound, message)
}

func (s *Server) notStoppableResponse(w http.ResponseWriter, r *http.Request) {
	message := "the server cannot be stopped, the host will be notified for the request"
	s.errorResponse(w, r, http.StatusForbidden, message)
}

func (s *Server) notIdleResponse(w http.ResponseWriter, r *http.Request) {
	message := "server is currently not idle (serving files)"
	s.errorResponse(w, r, http.StatusConflict, message)
}
