// adapters/in/rest/metrics.go — middleware Prometheus + endpoint /metrics
package rest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "myr",
		Name:      "http_requests_total",
		Help:      "Nombre total de requêtes HTTP par méthode, route et code de statut.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "myr",
		Name:      "http_request_duration_seconds",
		Help:      "Durée des requêtes HTTP en secondes.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	activeSessions = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "myr",
		Name:      "active_sessions",
		Help:      "Nombre de sessions actives (X-Myr-Token).",
	}, func() float64 { return 0 }) // remplacé par WithMetrics
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

// metricsHandler retourne le handler Prometheus pour /metrics.
func (h *Handler) metricsHandler() http.Handler {
	// Remplace la gauge statique par une vraie lecture du sessionStore
	prometheus.Unregister(activeSessions) //nolint:errcheck
	activeSessions = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "myr",
		Name:      "active_sessions",
		Help:      "Nombre de sessions actives (X-Myr-Token).",
	}, func() float64 {
		if h.sessions == nil {
			return 0
		}
		return float64(len(h.sessions.list()))
	})
	prometheus.MustRegister(activeSessions)
	return promhttp.Handler()
}

// responseRecorder capture le code de statut HTTP pour les métriques.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

// metricsMiddleware enregistre durée et compteur de chaque requête.
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rr, r)
		dur := time.Since(start).Seconds()

		path := normalizePath(r.URL.Path)
		status := strconv.Itoa(rr.status)

		httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(dur)
	})
}

// normalizePath remplace les segments UUIDs/IDs par "{id}" pour limiter la cardinalité.
func normalizePath(p string) string {
	// Règles simples : /api/components/abc123 → /api/components/{id}
	if len(p) < 5 {
		return p
	}
	// Cherche un segment final de plus de 8 caractères après /api/*/
	parts := splitPath(p)
	if len(parts) >= 3 {
		last := parts[len(parts)-1]
		if len(last) > 8 {
			parts[len(parts)-1] = "{id}"
			return joinPath(parts)
		}
	}
	return p
}

func splitPath(p string) []string {
	var parts []string
	cur := ""
	for _, c := range p {
		if c == '/' {
			if cur != "" {
				parts = append(parts, cur)
			}
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

func joinPath(parts []string) string {
	s := ""
	for _, p := range parts {
		s += "/" + p
	}
	return s
}
