// adapters/out/localstorage/request_store.go — persistance JSON des demandes de compte
package localstorage

import (
	"encoding/json"
	"os"

	"myr/domain/identity"
)

var _ identity.RequestStore = (*JSONRequestStore)(nil)

// JSONRequestStore persiste les demandes de compte dans un fichier JSON.
type JSONRequestStore struct {
	path string
}

func NewJSONRequestStore(path string) *JSONRequestStore {
	return &JSONRequestStore{path: path}
}

func (s *JSONRequestStore) load() ([]*identity.AccountRequest, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var list []*identity.AccountRequest
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *JSONRequestStore) save(list []*identity.AccountRequest) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *JSONRequestStore) Save(req *identity.AccountRequest) error {
	list, err := s.load()
	if err != nil {
		return err
	}
	// Remplacer si l'ID existe déjà, sinon ajouter
	for i, r := range list {
		if r.ID == req.ID {
			list[i] = req
			return s.save(list)
		}
	}
	list = append(list, req)
	return s.save(list)
}

func (s *JSONRequestStore) FindAll() ([]*identity.AccountRequest, error) {
	return s.load()
}
