// adapters/out/localstorage/network_store.go
package localstorage

import (
	"encoding/json"
	"os"

	"myr-core/domain/network"
)

var _ network.Repo = (*JSONNetworkStore)(nil)

// JSONNetworkStore persiste les profils réseau dans un fichier JSON.
type JSONNetworkStore struct {
	path string
}

func NewJSONNetworkStore(path string) *JSONNetworkStore {
	return &JSONNetworkStore{path: path}
}

func (s *JSONNetworkStore) load() ([]*network.NetworkProfile, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var list []*network.NetworkProfile
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *JSONNetworkStore) flush(list []*network.NetworkProfile) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *JSONNetworkStore) Save(n *network.NetworkProfile) error {
	list, err := s.load()
	if err != nil {
		return err
	}
	for i, existing := range list {
		if existing.ID == n.ID {
			list[i] = n
			return s.flush(list)
		}
	}
	list = append(list, n)
	return s.flush(list)
}

func (s *JSONNetworkStore) FindAll() ([]*network.NetworkProfile, error) {
	return s.load()
}

func (s *JSONNetworkStore) Delete(id string) error {
	list, err := s.load()
	if err != nil {
		return err
	}
	filtered := list[:0]
	for _, n := range list {
		if n.ID != id {
			filtered = append(filtered, n)
		}
	}
	return s.flush(filtered)
}
