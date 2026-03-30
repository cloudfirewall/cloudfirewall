package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/cloudfirewall/cloudfirewall/apps/api/internal/service"
	"github.com/cloudfirewall/cloudfirewall/apps/api/types"
)

type Server struct {
	store *service.Store
	mux   *http.ServeMux
}

func NewServer(store *service.Store) *Server {
	s := &Server{
		store: store,
		mux:   http.NewServeMux(),
	}

	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /swagger", s.handleSwaggerUI)
	s.mux.HandleFunc("GET /openapi.json", s.handleOpenAPI)
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("POST /api/v1/admin/login", s.handleAdminLogin)
	s.mux.HandleFunc("POST /api/v1/enrollment-tokens", s.handleCreateEnrollmentToken)
	s.mux.HandleFunc("POST /api/v1/firewall-config", s.handleUpdateFirewallConfig)
	s.mux.HandleFunc("POST /api/v1/enroll", s.handleEnroll)
	s.mux.HandleFunc("POST /api/v1/agents/self/heartbeat", s.handleHeartbeat)
	s.mux.HandleFunc("GET /api/v1/agents/self/config", s.handleConfig)
	s.mux.HandleFunc("GET /api/v1/agents", s.handleListAgents)
}

func (s *Server) handleSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Cloudfirewall API Docs</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: "/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
      });
    </script>
  </body>
</html>`))
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, openAPISpec())
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req types.AdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := s.store.AdminLogin(req)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreateEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	if err := s.authorizeAdminRequest(r); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req types.CreateEnrollmentTokenRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	resp, err := s.store.CreateEnrollmentToken(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleUpdateFirewallConfig(w http.ResponseWriter, r *http.Request) {
	if err := s.authorizeAdminRequest(r); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req types.UpdateFirewallConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := s.store.UpdateFirewallConfig(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	var req types.EnrollAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := s.store.Enroll(req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidEnrollmentToken) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	authToken, ok := bearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	var req types.AgentHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := s.store.Heartbeat(authToken, req)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	authToken, ok := bearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	resp, err := s.store.Config(authToken)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if err := s.authorizeAdminRequest(r); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s.store.ListAgents())
}

func (s *Server) authorizeAdminRequest(r *http.Request) error {
	if apiKey := strings.TrimSpace(r.Header.Get("X-API-Key")); apiKey != "" {
		return s.store.AuthorizeAPIKey(apiKey)
	}

	authToken, ok := bearerToken(r)
	if !ok {
		return service.ErrUnauthorized
	}

	return s.store.AuthorizeAdminSession(authToken)
}

func bearerToken(r *http.Request) (string, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", false
	}

	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return "", false
	}

	token = strings.TrimSpace(token)
	return token, token != ""
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
