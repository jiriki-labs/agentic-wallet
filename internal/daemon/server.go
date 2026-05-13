package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jiriki-labs/agentic-wallet/internal/audit"
	"github.com/jiriki-labs/agentic-wallet/internal/config"
	"github.com/jiriki-labs/agentic-wallet/internal/keystore"
	"github.com/jiriki-labs/agentic-wallet/internal/policy"
	"github.com/jiriki-labs/agentic-wallet/internal/x402"
)

// Server is the Jiriki wallet daemon.
type Server struct {
	signer         *keystore.Signer
	auditStore     *audit.Store
	policyEngine   *policy.Engine
	x402Client     x402.Client
	idemCache      *idempotencyCache
	confirmQ       *confirmQueue
	facilitatorURL string
	httpServer     *http.Server
}

// Config holds server startup configuration.
type Config struct {
	SocketPath     string
	ListenTCP      string // empty = use socket
	FacilitatorURL string
}

// New creates a new daemon Server.
func New(signer *keystore.Signer, auditStore *audit.Store, policyEngine *policy.Engine, cfg Config) *Server {
	facilitatorURL := cfg.FacilitatorURL
	client := x402.NewNativeClient()
	if facilitatorURL != "" {
		client.SetFacilitatorURL(facilitatorURL)
	}

	return &Server{
		signer:         signer,
		auditStore:     auditStore,
		policyEngine:   policyEngine,
		x402Client:     client,
		idemCache:      newIdempotencyCache(256, 5*time.Minute),
		confirmQ:       newConfirmQueue(),
		facilitatorURL: facilitatorURL,
	}
}

// Start starts the HTTP server on the configured transport.
func (s *Server) Start(cfg Config) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/balance", s.handleBalance)
	mux.HandleFunc("/policy", s.handlePolicy)
	mux.HandleFunc("/pay-x402", s.handlePayX402)
	mux.HandleFunc("/transactions", s.handleTransactions)
	mux.HandleFunc("/approve", s.handleApprove)

	handler := s.middlewareChain(mux, cfg)
	s.httpServer = &http.Server{Handler: handler}

	if cfg.ListenTCP != "" {
		ln, err := net.Listen("tcp", cfg.ListenTCP)
		if err != nil {
			return fmt.Errorf("listen TCP %s: %w", cfg.ListenTCP, err)
		}
		go func() { _ = s.httpServer.Serve(ln) }()
		fmt.Printf("jiriki daemon listening on TCP %s\n", cfg.ListenTCP)
		return nil
	}

	// Default: Unix socket
	socketPath := cfg.SocketPath
	if socketPath == "" {
		socketPath = config.SocketPath()
	}
	if err := os.MkdirAll(parentDir(socketPath), 0700); err != nil {
		return fmt.Errorf("mkdir socket dir: %w", err)
	}
	_ = os.Remove(socketPath) // clean up stale socket
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen Unix socket %s: %w", socketPath, err)
	}
	if err := os.Chmod(socketPath, 0600); err != nil {
		ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	go func() { _ = s.httpServer.Serve(ln) }()
	fmt.Printf("jiriki daemon listening on Unix socket %s\n", socketPath)
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.idemCache != nil {
		s.idemCache.stop()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) middlewareChain(next http.Handler, cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In TCP mode: validate loopback and check bearer token
		if cfg.ListenTCP != "" {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			ip := net.ParseIP(host)
			if ip == nil || !ip.IsLoopback() {
				http.Error(w, "forbidden: non-loopback source", http.StatusForbidden)
				return
			}
			// Check bearer token
			authHeader := r.Header.Get("Authorization")
			expectedToken := loadAuthToken()
			if expectedToken != "" {
				if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != expectedToken {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func loadAuthToken() string {
	data, err := os.ReadFile(config.AuthFile())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"address": s.signer.Address().Hex(),
		"note":    "use jiriki balance for live on-chain balance",
	})
}

func (s *Server) handlePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.policyEngine.Config())
}

func (s *Server) handleTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	records, err := s.auditStore.List("", time.Time{}, 50)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, records)
}

// parentDir returns the parent directory of path, or "." if none.
func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == os.PathSeparator {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}
