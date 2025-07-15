package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type envelop map[string]any

func (*Server) writeJSON(w http.ResponseWriter, data envelop, status int, headers http.Header) error {
	jsonBytes, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	for k, v := range headers {
		w.Header()[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(jsonBytes)
	return nil
}

func (s *Server) render(w http.ResponseWriter, status int, page string, data any) error {
	ts := getTemplate()
	if ts == nil {
		return fmt.Errorf("template doesnot exist %q", page)
	}
	b := new(bytes.Buffer)
	if err := ts.ExecuteTemplate(b, "fileIndexes", data); err != nil {
		return err
	}
	w.WriteHeader(status)
	if _, err := b.WriteTo(w); err != nil {
		return fmt.Errorf("writing response: %w", err)
	}
	return nil
}
