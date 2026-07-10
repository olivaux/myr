// adapters/out/fabric/channel_store.go
package fabric

import (
	"encoding/json"
	"fmt"

	"myr/domain/channel"
)

// FabricChannelStore lit les canaux disponibles depuis le ledger (lecture seule).
// En pratique, remplacé par StaticChannelStore initialisé depuis le profil de connexion.
type FabricChannelStore struct {
	gc *GatewayClient
}

func NewFabricChannelStore(gc *GatewayClient) *FabricChannelStore {
	return &FabricChannelStore{gc: gc}
}

// FindByID récupère un canal par son ID depuis le ledger.
func (f *FabricChannelStore) FindByID(id string) (*channel.Channel, error) {
	result, err := f.gc.Contract().EvaluateTransaction("GetChannel", id)
	if err != nil {
		return nil, fmt.Errorf("fabric GetChannel evaluate : %w", err)
	}
	if len(result) == 0 {
		return nil, nil
	}
	var ch channel.Channel
	if err := json.Unmarshal(result, &ch); err != nil {
		return nil, fmt.Errorf("fabric GetChannel unmarshal : %w", err)
	}
	return &ch, nil
}

// FindAll liste tous les canaux depuis le ledger.
func (f *FabricChannelStore) FindAll() ([]*channel.Channel, error) {
	result, err := f.gc.Contract().EvaluateTransaction("ListChannels")
	if err != nil {
		return nil, fmt.Errorf("fabric ListChannels evaluate : %w", err)
	}
	var list []*channel.Channel
	if err := json.Unmarshal(result, &list); err != nil {
		return nil, fmt.Errorf("fabric ListChannels unmarshal : %w", err)
	}
	return list, nil
}
