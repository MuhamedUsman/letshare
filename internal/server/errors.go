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

func (s *Server) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	s.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

func (s *Server) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource cannot be found"
	s.errorResponse(w, r, http.StatusNotFound, message)
}

func (s *Server) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource cannot be modified due to edit conflicts, please try again"
	s.errorResponse(w, r, http.StatusConflict, message)
}

func (s *Server) alreadyActivatedResponse(w http.ResponseWriter, r *http.Request) {
	message := "account is already active"
	s.errorResponse(w, r, http.StatusConflict, message)
}

func (s *Server) invalidCredentialResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	s.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (s *Server) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	s.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (s *Server) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	s.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (s *Server) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	s.errorResponse(w, r, http.StatusForbidden, message)
}

func (s *Server) redundantSubscription(w http.ResponseWriter, r *http.Request) {
	message := "single instance of subscription is allowed for this account"
	s.errorResponse(w, r, http.StatusConflict, message)
}
