// adapters/in/rest/session.go — gestion des sessions (tokens) côté serveur
package rest

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// sessionTTL : 7 jours — survit aux redémarrages de l'app.
const sessionTTL = 7 * 24 * time.Hour

// myrSession représente une session active associée à un token opaque.
type myrSession struct {
	Token     string    `json:"token"`
	Role      string    `json:"role"`                 // "reader" | "contributor" | "admin"
	Pseudo    string    `json:"pseudo"`
	Channel   string    `json:"channel"`              // canal Fabric actif
	NetworkID string    `json:"network_id,omitempty"` // réseau Fabric actif (multi-réseau)
	ExpiresAt time.Time `json:"expires_at"`
}

// sessionBackend est l'interface du store de sessions — implémentée par sessionStore (mémoire)
// et redisSessionStore (Redis).
type sessionBackend interface {
	create(pseudo, role, channel string) (*myrSession, error)
	get(token string) *myrSession
	setNetwork(token, networkID string) bool
	setChannel(token, channel string) bool
	delete(token string)
	list() []*myrSession
}

// sessionStore est un store de sessions en mémoire avec persistance optionnelle sur disque.
type sessionStore struct {
	mu       sync.RWMutex
	data     map[string]*myrSession
	filePath string
}

func newSessionStore() *sessionStore {
	return &sessionStore{data: make(map[string]*myrSession)}
}

// withPersistence active la sauvegarde/restauration des sessions dans un fichier JSON.
// Les sessions expirées ne sont pas rechargées.
func (s *sessionStore) withPersistence(path string) {
	s.filePath = path
	s.loadFromFile()
}

func (s *sessionStore) loadFromFile() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}
	var sessions []*myrSession
	if json.Unmarshal(data, &sessions) != nil {
		return
	}
	now := time.Now()
	s.mu.Lock()
	for _, sess := range sessions {
		if now.Before(sess.ExpiresAt) {
			s.data[sess.Token] = sess
		}
	}
	s.mu.Unlock()
}

func (s *sessionStore) saveToFile() {
	if s.filePath == "" {
		return
	}
	s.mu.RLock()
	now := time.Now()
	sessions := make([]*myrSession, 0, len(s.data))
	for _, sess := range s.data {
		if now.Before(sess.ExpiresAt) {
			sessions = append(sessions, sess)
		}
	}
	s.mu.RUnlock()
	data, _ := json.MarshalIndent(sessions, "", "  ")
	os.WriteFile(s.filePath, data, 0600) //nolint:errcheck
}

func (s *sessionStore) create(pseudo, role, channel string) (*myrSession, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("génération token impossible : %w", err)
	}
	sess := &myrSession{
		Token:     hex.EncodeToString(b),
		Role:      role,
		Pseudo:    pseudo,
		Channel:   channel,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	s.mu.Lock()
	s.data[sess.Token] = sess
	s.mu.Unlock()
	s.saveToFile()
	return sess, nil
}

// setNetwork met à jour le réseau actif d'une session existante.
func (s *sessionStore) setNetwork(token, networkID string) bool {
	s.mu.Lock()
	sess := s.data[token]
	if sess == nil || time.Now().After(sess.ExpiresAt) {
		s.mu.Unlock()
		return false
	}
	sess.NetworkID = networkID
	s.mu.Unlock()
	s.saveToFile()
	return true
}

// setChannel met à jour le canal actif d'une session existante.
func (s *sessionStore) setChannel(token, channel string) bool {
	s.mu.Lock()
	sess := s.data[token]
	if sess == nil || time.Now().After(sess.ExpiresAt) {
		s.mu.Unlock()
		return false
	}
	sess.Channel = channel
	s.mu.Unlock()
	s.saveToFile()
	return true
}

// get retourne la session si le token est valide et non expiré, nil sinon.
// La comparaison finale en temps constant prévient les attaques par timing.
func (s *sessionStore) get(token string) *myrSession {
	if token == "" {
		return nil
	}
	s.mu.RLock()
	sess := s.data[token]
	s.mu.RUnlock()
	if sess == nil || time.Now().After(sess.ExpiresAt) {
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(sess.Token), []byte(token)) != 1 {
		return nil
	}
	return sess
}

// delete supprime une session (révocation immédiate).
func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	delete(s.data, token)
	s.mu.Unlock()
	s.saveToFile()
}

// list retourne toutes les sessions actives (pour l'API admin).
func (s *sessionStore) list() []*myrSession {
	now := time.Now()
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*myrSession, 0, len(s.data))
	for _, sess := range s.data {
		if now.Before(sess.ExpiresAt) {
			out = append(out, sess)
		}
	}
	return out
}

// ── Clé de contexte ──────────────────────────────────────────────────────────

type ctxKey int

const (
	ctxSession ctxKey = iota
)

// sessionFromCtx récupère la session depuis le contexte de la requête.
func sessionFromCtx(r *http.Request) *myrSession {
	v := r.Context().Value(ctxSession)
	if v == nil {
		return nil
	}
	s, _ := v.(*myrSession)
	return s
}
