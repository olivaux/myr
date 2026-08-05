// adapters/in/rest/server.go — adaptateur entrant HTTP (API REST uniquement)
package rest

import (
	"context"
	"log"
	"net/http"
	"time"

	rbac "myr-core/domain/role"
)

// Server expose l'API REST sur un port local. Aucune interface graphique
// n'est servie ici — le GUI est un dépôt externe, client de cette API.
type Server struct {
	handler *Handler
	addr    string
}

func NewServer(h *Handler, addr string) *Server {
	return &Server{handler: h, addr: addr}
}

// secureHeaders ajoute les en-têtes de sécurité HTTP à chaque réponse.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		next.ServeHTTP(w, r)
	})
}

// Handler construit et retourne le http.Handler complet de l'API REST.
func (s *Server) Handler() (http.Handler, error) {
	return s.buildMux()
}

// Start démarre le serveur et bloque jusqu'à une erreur (usage dev/simple).
func (s *Server) Start() error {
	mux, err := s.buildMux()
	if err != nil {
		return err
	}
	log.Printf("Serveur REST démarré sur http://%s\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// StartWithShutdown démarre le serveur et se termine proprement quand ctx est annulé.
// À utiliser en production avec signal.NotifyContext.
func (s *Server) StartWithShutdown(ctx context.Context) error {
	mux, err := s.buildMux()
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}
	log.Printf("Serveur REST démarré sur http://%s\n", s.addr)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		log.Println("Arrêt propre du serveur HTTP…")
		shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if shutErr := srv.Shutdown(shutCtx); shutErr != nil {
			log.Printf("Erreur shutdown : %v", shutErr)
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (s *Server) buildMux() (http.Handler, error) {
	mux := http.NewServeMux()

	// Routes d'identité — ouvertes (point d'entrée, pas d'auth requise)
	mux.HandleFunc("/api/identity/policy", s.handler.handleIdentityPolicy)
	mux.HandleFunc("/api/identity/enroll", s.handler.handleIdentityEnroll)
	mux.HandleFunc("/api/identity/session", s.handler.handleIdentitySession)
	mux.HandleFunc("/api/identity/guest", s.handler.handleIdentityGuest)
	mux.HandleFunc("/api/identity/request", s.handler.handleIdentityRequest)

	// Routes d'identité — protégées
	mux.HandleFunc("/api/identity/requests", s.handler.requireRole(rbac.PermIdentityAdmin, s.handler.handleIdentityRequests))
	mux.HandleFunc("/api/identity/wallets", s.handler.requireAuth(s.handler.handleIdentityWallets))
	mux.HandleFunc("/api/identity/status", s.handler.requireAuth(s.handler.handleIdentityStatus))

	// Routes de données — lecture : reader+, écriture (POST/PUT/PATCH/DELETE) : contributor+
	auth := s.handler.requireAuth
	contrib := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				s.handler.requireRole(rbac.PermWrite, next)(w, r)
			default:
				auth(next)(w, r)
			}
		}
	}
	mux.HandleFunc("/api/components", contrib(s.handler.handleComponents))
	mux.HandleFunc("/api/components/", contrib(s.handler.handleComponent))
	mux.HandleFunc("/api/connections", contrib(s.handler.handleConnections))
	mux.HandleFunc("/api/connections/", contrib(s.handler.handleConnection))
	mux.HandleFunc("/api/assembly-links", contrib(s.handler.handleAssemblyLinks))
	mux.HandleFunc("/api/virtual-connect", contrib(s.handler.handleVirtualConnect))
	mux.HandleFunc("/api/modules", contrib(s.handler.handleModules))
	mux.HandleFunc("/api/modules/", contrib(s.handler.handleModule))
	mux.HandleFunc("/api/interfaces/", contrib(s.handler.handleInterface))
	mux.HandleFunc("/api/refs", auth(s.handler.handleRefs))
	mux.HandleFunc("/api/refs/categories", auth(s.handler.handleRefCategories))
	mux.HandleFunc("/api/refs/types", auth(s.handler.handleRefTypes))
	mux.HandleFunc("/api/refs/units", auth(s.handler.handleRefUnits))
	mux.HandleFunc("/api/channels", s.handler.handleChannels)             // GET public, PUT protégé
	mux.HandleFunc("/api/networks", s.handler.handleNetworks)             // GET liste des réseaux
	mux.HandleFunc("/api/networks/active", s.handler.handleNetworkActive) // GET/PUT réseau actif session
	mux.HandleFunc("/api/ping", s.handler.handlePing)                     // liveness — pas d'appel Fabric
	mux.HandleFunc("/api/status", s.handler.handleStatus)                 // status toujours accessible
	mux.HandleFunc("/api/licenses", contrib(s.handler.handleLicenses))
	mux.HandleFunc("/api/licenses/", contrib(s.handler.handleLicense))

	// Health + métriques — non authentifiés (appelés par infra/load-balancer)
	mux.HandleFunc("/api/health", s.handler.handleHealth)
	mux.Handle("/metrics", s.handler.metricsHandler())

	// Admin — réservé au rôle "admin"
	adminOnly := func(next http.HandlerFunc) http.HandlerFunc {
		return s.handler.requireRole(rbac.PermAdmin, next)
	}
	mux.HandleFunc("/api/admin/sessions", adminOnly(s.handler.handleAdminSessions))
	mux.HandleFunc("/api/admin/sessions/", adminOnly(s.handler.handleAdminSession))

	return secureHeaders(mux), nil
}
