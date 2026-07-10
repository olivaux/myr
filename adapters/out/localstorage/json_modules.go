// adapters/out/localstorage/json_modules.go
// Implémente model.BlockchainPort via un fichier JSON local.
// Utilisé comme backend de remplacement quand Fabric n'est pas disponible,
// garantissant la persistance des modules (brouillons inclus) entre les redémarrages.
package localstorage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"myr/domain/model"
)

var _ model.BlockchainPort = (*JSONModuleStore)(nil)

type moduleDB struct {
	Modules []*model.Model3D `json:"modules"`
}

// JSONModuleStore persiste les modules localement dans un fichier JSON.
// Thread-safe. Compatible avec BlockchainPort + l'interface optionnelle RemoveModelRecord.
type JSONModuleStore struct {
	path string
	mu   sync.RWMutex
}

func NewJSONModuleStore(path string) *JSONModuleStore {
	return &JSONModuleStore{path: path}
}

// ── persistance interne ──────────────────────────────────────────────────────

func (s *JSONModuleStore) load() *moduleDB {
	var db moduleDB
	data, err := os.ReadFile(s.path)
	if err == nil {
		json.Unmarshal(data, &db) //nolint:errcheck — fichier corrompu traité comme vide
	}
	if db.Modules == nil {
		db.Modules = []*model.Model3D{}
	}
	return &db
}

func (s *JSONModuleStore) save(db *moduleDB) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("json_modules marshal : %w", err)
	}
	return os.WriteFile(s.path, data, 0644)
}

// ── model.BlockchainPort ─────────────────────────────────────────────────────

// StoreModelRecord persiste ou met à jour un module (upsert par ID).
func (s *JSONModuleStore) StoreModelRecord(m *model.Model3D) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	db := s.load()
	for i, existing := range db.Modules {
		if existing.ID == m.ID {
			db.Modules[i] = m
			return s.save(db)
		}
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	db.Modules = append(db.Modules, m)
	return s.save(db)
}

// GetModelRecord retourne un module par ID. channelID est ignoré (stockage local non segmenté).
func (s *JSONModuleStore) GetModelRecord(id, _ string) (*model.Model3D, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	db := s.load()
	for _, m := range db.Modules {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, fmt.Errorf("module %q introuvable dans le store local", id)
}

// ListModelRecords retourne tous les modules. channelID est ignoré.
func (s *JSONModuleStore) ListModelRecords(_ string) ([]*model.Model3D, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	db := s.load()
	return db.Modules, nil
}

// VerifyIntegrity n'est pas supporté en mode local (pas de hachage blockchain).
func (s *JSONModuleStore) VerifyIntegrity(_, _, _ string) (bool, error) {
	return false, fmt.Errorf("vérification d'intégrité non disponible en mode stockage local")
}

// ── interface optionnelle RemoveModelRecord ──────────────────────────────────

// RemoveModelRecord supprime un module par ID. Appelé par Service.RemoveModule.
func (s *JSONModuleStore) RemoveModelRecord(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	db := s.load()
	filtered := db.Modules[:0]
	found := false
	for _, m := range db.Modules {
		if m.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, m)
	}
	if !found {
		return fmt.Errorf("module %q introuvable", id)
	}
	db.Modules = filtered
	return s.save(db)
}
