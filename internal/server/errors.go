package server

import (
	"net/http"
)

func (s *Server) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	data := envelop{"errors": message}
	preferJSON := r.Header.Get("Accept") == "application/json"
	if preferJSON {
		if err := s.writeJSON(w, data, status, nil); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(status)
	if _, err := w.Write([]byte(message.(string))); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) serverErrorResponse(w http.ResponseWriter, r *http.Request) {
	message := "the server encountered a problem and could not process your request"
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

func (s *Server) noOSHostnameAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := "Wrong door! Use my advertised mDNS service, not my OS hostname. This teapot has standards!"
	s.errorResponse(w, r, http.StatusTeapot, message)
}
