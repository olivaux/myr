// adapters/out/localstorage/channel_store.go
package localstorage

import (
	"encoding/json"
	"fmt"
	"os"

	"myr/domain/channel"
)

var (
	_ channel.ChannelRepo = (*JSONChannelStore)(nil)
	_ channel.ChannelRepo = (*StaticChannelStore)(nil)
)

// JSONChannelStore lit les canaux depuis un fichier JSON (lecture seule).
// Utilisé en mode dev pour simuler un ensemble de canaux sans réseau Fabric.
type JSONChannelStore struct {
	path string
}

func NewJSONChannelStore(path string) *JSONChannelStore {
	return &JSONChannelStore{path: path}
}

func (s *JSONChannelStore) load() (map[string]*channel.Channel, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return make(map[string]*channel.Channel), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read channel store: %w", err)
	}
	var m map[string]*channel.Channel
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse channel store: %w", err)
	}
	return m, nil
}

func (s *JSONChannelStore) FindByID(id string) (*channel.Channel, error) {
	m, err := s.load()
	if err != nil {
		return nil, err
	}
	return m[id], nil
}

func (s *JSONChannelStore) FindAll() ([]*channel.Channel, error) {
	m, err := s.load()
	if err != nil {
		return nil, err
	}
	list := make([]*channel.Channel, 0, len(m))
	for _, ch := range m {
		list = append(list, ch)
	}
	return list, nil
}

// StaticChannelStore retourne une liste fixe de canaux (en mémoire).
// Utilisé en mode Fabric : les canaux viennent du profil de connexion importé.
type StaticChannelStore struct {
	channels []*channel.Channel
	byID     map[string]*channel.Channel
}

func NewStaticChannelStore(channels []*channel.Channel) *StaticChannelStore {
	byID := make(map[string]*channel.Channel, len(channels))
	for _, ch := range channels {
		byID[ch.ID] = ch
	}
	return &StaticChannelStore{channels: channels, byID: byID}
}

func (s *StaticChannelStore) FindByID(id string) (*channel.Channel, error) {
	return s.byID[id], nil
}

func (s *StaticChannelStore) FindAll() ([]*channel.Channel, error) {
	return s.channels, nil
}
