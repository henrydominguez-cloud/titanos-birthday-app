// Package api wires up the HTTP layer: routing, validation and JSON responses.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/titanos/birthday-app/internal/birthday"
	"github.com/titanos/birthday-app/internal/store"
)

// username must be letters only (from the spec).
var usernameRe = regexp.MustCompile(`^[A-Za-z]+$`)

type Server struct {
	store store.Store
	log   *slog.Logger
	now   func() time.Time // injectable so tests can freeze time
}

func NewServer(s store.Store, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{store: s, log: logger, now: time.Now}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /hello/{username}", s.handlePut)
	mux.HandleFunc("GET /hello/{username}", s.handleGet)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	return recoverMiddleware(s.log, logMiddleware(s.log, mux))
}

type putRequest struct {
	DateOfBirth string `json:"dateOfBirth"`
	DateOfBrith string `json:"dateOfBrith"` // spec spells it this way; accept both
}

func (r putRequest) date() string {
	if r.DateOfBirth != "" {
		return r.DateOfBirth
	}
	return r.DateOfBrith
}

// PUT /hello/<username> {"dateOfBirth":"YYYY-MM-DD"} -> 204
func (s *Server) handlePut(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if !usernameRe.MatchString(username) {
		writeError(w, http.StatusBadRequest, "username must contain only letters")
		return
	}

	var req putRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	dob, err := time.ParseInLocation(birthday.Layout, req.date(), time.UTC)
	if err != nil {
		writeError(w, http.StatusBadRequest, "dateOfBirth must be a valid YYYY-MM-DD date")
		return
	}
	if !dob.Before(s.today()) {
		writeError(w, http.StatusBadRequest, "dateOfBirth must be a date before today")
		return
	}

	if err := s.store.Upsert(r.Context(), username, dob); err != nil {
		s.log.Error("upsert failed", "username", username, "err", err)
		writeError(w, http.StatusInternalServerError, "could not store user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type messageResponse struct {
	Message string `json:"message"`
}

// GET /hello/<username> -> 200 {"message": "..."}
func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if !usernameRe.MatchString(username) {
		writeError(w, http.StatusBadRequest, "username must contain only letters")
		return
	}

	dob, found, err := s.store.Get(r.Context(), username)
	if err != nil {
		s.log.Error("get failed", "username", username, "err", err)
		writeError(w, http.StatusInternalServerError, "could not read user")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{
		Message: birthday.Message(username, dob, s.now()),
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz checks the database so a pod only gets traffic when it can serve it.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "storage not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// today is the current date in UTC.
func (s *Server) today() time.Time {
	n := s.now().UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
