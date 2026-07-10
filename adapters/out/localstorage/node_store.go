// adapters/out/localstorage/node_store.go
// Bookkeeping local des nœuds (peer/orderer) provisionnés par myr sur un canal.
// Nécessaire car Hyperledger Fabric ne trace pas les peers individuellement dans
// la configuration de canal (seules les organisations y figurent) — seul myr sait
// quels nœuds il a provisionnés, information utilisée pour appliquer le seuil
// minimum de nœuds actifs (RM27) et pour retrouver un nœud à retirer.
package localstorage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// ProvisionedNode décrit un nœud (peer ou orderer) provisionné sur un canal donné.
type ProvisionedNode struct {
	ChannelID string    `json:"channel_id"`
	Type      string    `json:"type"` // "peer" | "orderer"
	Addr      string    `json:"addr"`
	OrgMSP    string    `json:"org_msp"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type nodeDB struct {
	Nodes []*ProvisionedNode `json:"nodes"`
}

// JSONNodeStore persiste les nœuds provisionnés dans un fichier JSON. Thread-safe.
type JSONNodeStore struct {
	path string
	mu   sync.RWMutex
}

func NewJSONNodeStore(path string) *JSONNodeStore {
	return &JSONNodeStore{path: path}
}

func (s *JSONNodeStore) load() *nodeDB {
	var db nodeDB
	data, err := os.ReadFile(s.path)
	if err == nil {
		json.Unmarshal(data, &db) //nolint:errcheck — fichier corrompu traité comme vide
	}
	if db.Nodes == nil {
		db.Nodes = []*ProvisionedNode{}
	}
	return &db
}

func (s *JSONNodeStore) save(db *nodeDB) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("node_store marshal : %w", err)
	}
	return os.WriteFile(s.path, data, 0644)
}

// Add enregistre un nœud (upsert par ChannelID+Addr).
func (s *JSONNodeStore) Add(n *ProvisionedNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	db := s.load()
	for i, existing := range db.Nodes {
		if existing.ChannelID == n.ChannelID && existing.Addr == n.Addr {
			db.Nodes[i] = n
			return s.save(db)
		}
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	db.Nodes = append(db.Nodes, n)
	return s.save(db)
}

// Deactivate marque un nœud comme inactif (retiré du canal) sans supprimer son historique.
func (s *JSONNodeStore) Deactivate(channelID, addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	db := s.load()
	for _, n := range db.Nodes {
		if n.ChannelID == channelID && n.Addr == addr {
			n.Active = false
			return s.save(db)
		}
	}
	return fmt.Errorf("nœud %s introuvable sur le canal %s", addr, channelID)
}

// Find retourne le nœud correspondant, ou nil si absent.
func (s *JSONNodeStore) Find(channelID, addr string) (*ProvisionedNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	db := s.load()
	for _, n := range db.Nodes {
		if n.ChannelID == channelID && n.Addr == addr {
			return n, nil
		}
	}
	return nil, nil
}

// ActiveCount retourne le nombre de nœuds actifs sur un canal.
func (s *JSONNodeStore) ActiveCount(channelID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	db := s.load()
	count := 0
	for _, n := range db.Nodes {
		if n.ChannelID == channelID && n.Active {
			count++
		}
	}
	return count, nil
}

// ListActive retourne les nœuds actifs d'un canal, filtrés par type si fourni ("" = tous).
func (s *JSONNodeStore) ListActive(channelID, nodeType string) ([]*ProvisionedNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	db := s.load()
	var out []*ProvisionedNode
	for _, n := range db.Nodes {
		if n.ChannelID == channelID && n.Active && (nodeType == "" || n.Type == nodeType) {
			out = append(out, n)
		}
	}
	return out, nil
}
