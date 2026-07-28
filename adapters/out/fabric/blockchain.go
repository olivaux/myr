// adapters/out/fabric/blockchain.go
package fabric

import (
	"encoding/json"
	"fmt"
	"regexp"

	"myr-core/domain/model"
)

var validID = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]{1,128}$`)

func validateID(v, label string) error {
	if !validID.MatchString(v) {
		return fmt.Errorf("%s invalide : caractères non autorisés ou longueur hors limites", label)
	}
	return nil
}

var _ model.BlockchainPort = (*FabricBlockchain)(nil)

// FabricBlockchain implémente model.BlockchainPort via le chaincode myrcc.
type FabricBlockchain struct {
	gc GatewayProvider
}

func NewFabricBlockchain(gc GatewayProvider) *FabricBlockchain {
	return &FabricBlockchain{gc: gc}
}

// StoreModelRecord soumet une transaction sur le canal de l'asset (m.ChannelID).
func (f *FabricBlockchain) StoreModelRecord(m *model.Model3D) error {
	if err := validateID(m.ChannelID, "channelID"); err != nil {
		return err
	}
	if err := validateID(m.ID, "model ID"); err != nil {
		return err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("fabric StoreModelRecord marshal : %w", err)
	}
	if _, err := f.gc.ContractFor(m.ChannelID).SubmitTransaction("StoreModel", string(data)); err != nil {
		return fmt.Errorf("fabric StoreModelRecord submit : %w", err)
	}
	return nil
}

// GetModelRecord évalue une query sur le canal spécifié.
// channelID="" utilise le canal par défaut de la configuration.
func (f *FabricBlockchain) GetModelRecord(id, channelID string) (*model.Model3D, error) {
	if err := validateID(id, "model ID"); err != nil {
		return nil, err
	}
	result, err := f.gc.ContractFor(channelID).EvaluateTransaction("GetModel", id)
	if err != nil {
		return nil, fmt.Errorf("fabric GetModelRecord evaluate : %w", err)
	}
	var m model.Model3D
	if err := json.Unmarshal(result, &m); err != nil {
		return nil, fmt.Errorf("fabric GetModelRecord unmarshal : %w", err)
	}
	return &m, nil
}

// ListModelRecords évalue une query sur le canal spécifié.
func (f *FabricBlockchain) ListModelRecords(channelID string) ([]*model.Model3D, error) {
	if err := validateID(channelID, "channelID"); err != nil {
		return nil, err
	}
	result, err := f.gc.ContractFor(channelID).EvaluateTransaction("ListModels", channelID)
	if err != nil {
		return nil, fmt.Errorf("fabric ListModelRecords evaluate : %w", err)
	}
	var list []*model.Model3D
	if err := json.Unmarshal(result, &list); err != nil {
		return nil, fmt.Errorf("fabric ListModelRecords unmarshal : %w", err)
	}
	return list, nil
}

// VerifyIntegrity vérifie l'intégrité d'un modèle sur le canal spécifié.
// channelID="" utilise le canal par défaut de la configuration.
func (f *FabricBlockchain) VerifyIntegrity(id, hash, channelID string) (bool, error) {
	if err := validateID(id, "model ID"); err != nil {
		return false, err
	}
	result, err := f.gc.ContractFor(channelID).EvaluateTransaction("VerifyModel", id, hash)
	if err != nil {
		return false, fmt.Errorf("fabric VerifyIntegrity evaluate : %w", err)
	}
	return string(result) == "true", nil
}
