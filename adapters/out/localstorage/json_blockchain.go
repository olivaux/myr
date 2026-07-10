// adapters/out/localstorage/json_blockchain.go
// Implémente ConnectionStore + ThumbnailStore + InterfaceStore
// via un fichier JSON local. Stocke les données GUI locales (connexions,
// miniatures, interfaces) indépendamment de la blockchain Fabric.
package localstorage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"myr/domain/model"
)

type dbFile struct {
	Thumbnails  map[string]string       `json:"thumbnails"`
	Connections []*model.Connection     `json:"connections"`
	Interfaces  []*model.AssetInterface `json:"interfaces"`
	Refs        *model.InterfaceRefs    `json:"refs"`
}

// JSONBlockchain persiste les données GUI locales (connexions, miniatures, interfaces).
// Il implémente ConnectionStore, ThumbnailStore et InterfaceStore.
type JSONBlockchain struct {
	path string
	mu   sync.RWMutex
}

func NewJSONBlockchain(path string) *JSONBlockchain {
	return &JSONBlockchain{path: path}
}

// ── persistance interne ──────────────────────────────────────────────────────

func (j *JSONBlockchain) load() *dbFile {
	data, err := os.ReadFile(j.path)
	var db dbFile
	if err == nil {
		if unmarshalErr := json.Unmarshal(data, &db); unmarshalErr != nil {
			log.Printf("[localstorage] fichier DB corrompu (%s), démarrage à vide : %v", j.path, unmarshalErr)
		}
	}
	// Initialiser tous les champs nuls pour éviter les panics sur les callers.
	if db.Thumbnails == nil {
		db.Thumbnails = map[string]string{}
	}
	if db.Connections == nil {
		db.Connections = []*model.Connection{}
	}
	if db.Interfaces == nil {
		db.Interfaces = []*model.AssetInterface{}
	}
	// Refs : appliquer les valeurs par défaut si absentes ou sans catégories.
	if db.Refs == nil || len(db.Refs.Categories) == 0 {
		db.Refs = model.DefaultInterfaceRefs()
	}
	return &db
}

func (j *JSONBlockchain) save(db *dbFile) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(j.path, data, 0644)
}

// ── model.ConnectionStore ────────────────────────────────────────────────────

func (j *JSONBlockchain) SaveConnection(c *model.Connection) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	db := j.load()

	// Dédoublonnage : même paire d'interfaces ET même paire d'instances
	// (les mêmes interfaces peuvent être reconnectées dans un autre produit/contexte)
	for _, existing := range db.Connections {
		if c.FromIfaceID != "" && c.ToIfaceID != "" {
			if existing.FromIfaceID == c.FromIfaceID && existing.ToIfaceID == c.ToIfaceID &&
				existing.FromInstanceID == c.FromInstanceID && existing.ToInstanceID == c.ToInstanceID {
				return nil
			}
		} else {
			if existing.From == c.From && existing.To == c.To {
				return nil
			}
		}
	}
	db.Connections = append(db.Connections, c)
	return j.save(db)
}

func (j *JSONBlockchain) UpdateConnection(c *model.Connection) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	db := j.load()
	for i, existing := range db.Connections {
		if existing.ID == c.ID {
			db.Connections[i] = c
			return j.save(db)
		}
	}
	return fmt.Errorf("connexion %s introuvable", c.ID)
}

func (j *JSONBlockchain) RemoveConnection(id string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	db := j.load()
	filtered := db.Connections[:0]
	for _, c := range db.Connections {
		if c.ID != id {
			filtered = append(filtered, c)
		}
	}
	db.Connections = filtered
	return j.save(db)
}

func (j *JSONBlockchain) ListConnections() ([]*model.Connection, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	db := j.load()
	return db.Connections, nil
}

// ── model.ThumbnailStore ─────────────────────────────────────────────────────

func (j *JSONBlockchain) SaveThumbnail(assetID, dataURL string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	db := j.load()
	db.Thumbnails[assetID] = dataURL
	return j.save(db)
}

func (j *JSONBlockchain) GetThumbnail(assetID string) (string, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	db := j.load()
	return db.Thumbnails[assetID], nil
}

// ── model.InterfaceStore ─────────────────────────────────────────────────────

func (j *JSONBlockchain) SaveInterface(iface *model.AssetInterface) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	db := j.load()
	for i, existing := range db.Interfaces {
		if existing.ID == iface.ID {
			db.Interfaces[i] = iface
			return j.save(db)
		}
	}
	db.Interfaces = append(db.Interfaces, iface)
	return j.save(db)
}

func (j *JSONBlockchain) RemoveInterface(id string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	db := j.load()
	filtered := db.Interfaces[:0]
	for _, iface := range db.Interfaces {
		if iface.ID != id {
			filtered = append(filtered, iface)
		}
	}
	db.Interfaces = filtered
	return j.save(db)
}

func (j *JSONBlockchain) ListInterfacesForAsset(assetID string) ([]*model.AssetInterface, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	db := j.load()
	var result []*model.AssetInterface
	for _, iface := range db.Interfaces {
		if iface.AssetID == assetID {
			result = append(result, iface)
		}
	}
	return result, nil
}

func (j *JSONBlockchain) GetInterface(id string) (*model.AssetInterface, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	db := j.load()
	for _, iface := range db.Interfaces {
		if iface.ID == id {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("interface %q introuvable", id)
}

func (j *JSONBlockchain) GetRefs() (*model.InterfaceRefs, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.load().Refs, nil // toujours non-nil grâce à load()
}

func (j *JSONBlockchain) AddRefCategory(cat string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	db := j.load()
	for _, c := range db.Refs.Categories {
		if c == cat {
			return nil
		}
	}
	db.Refs.Categories = append(db.Refs.Categories, cat)
	if db.Refs.Types == nil {
		db.Refs.Types = map[string][]string{}
	}
	if db.Refs.Units == nil {
		db.Refs.Units = map[string][]string{}
	}
	if _, ok := db.Refs.Types[cat]; !ok {
		db.Refs.Types[cat] = []string{}
	}
	if _, ok := db.Refs.Units[cat]; !ok {
		db.Refs.Units[cat] = []string{}
	}
	return j.save(db)
}

func (j *JSONBlockchain) AddRefType(cat, typeName string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	db := j.load()
	for _, t := range db.Refs.Types[cat] {
		if t == typeName {
			return nil
		}
	}
	db.Refs.Types[cat] = append(db.Refs.Types[cat], typeName)
	return j.save(db)
}

func (j *JSONBlockchain) AddRefUnit(cat, unit string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	db := j.load()
	for _, u := range db.Refs.Units[cat] {
		if u == unit {
			return nil
		}
	}
	db.Refs.Units[cat] = append(db.Refs.Units[cat], unit)
	return j.save(db)
}
