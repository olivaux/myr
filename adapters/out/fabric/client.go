// adapters/out/fabric/client.go — implémentation Fabric
package fabric

import (
	"encoding/json"
	"myr-core/domain/model"
)

// #incoherence — duplique le même port que FabricBlockchain (blockchain.go,
// celui réellement câblé dans cmd/api/main.go), mais avale les erreurs de
// (dé)sérialisation JSON au lieu de les remonter, et n'est construit nulle
// part hors des tests du paquet — voir specs/roadmap_dev.md § Écarts — revue
// de code, E5.
type FabricAdapter struct {
	contract ContractCaller
}

func NewFabricAdapter(contract ContractCaller) *FabricAdapter {
	return &FabricAdapter{contract: contract}
}

func (f *FabricAdapter) StoreModelRecord(m *model.Model3D) error {
	data, _ := json.Marshal(m)
	_, err := f.contract.SubmitTransaction("CreateModel", string(data))
	return err
}

func (f *FabricAdapter) GetModelRecord(id, _ string) (*model.Model3D, error) {
	result, err := f.contract.EvaluateTransaction("GetModel", id)
	if err != nil {
		return nil, err
	}
	var m model.Model3D
	json.Unmarshal(result, &m)
	return &m, nil
}

func (f *FabricAdapter) ListModelRecords(channelID string) ([]*model.Model3D, error) {
	result, err := f.contract.EvaluateTransaction("ListModels", channelID)
	if err != nil {
		return nil, err
	}
	var list []*model.Model3D
	json.Unmarshal(result, &list)
	return list, nil
}

func (f *FabricAdapter) VerifyIntegrity(id, hash, _ string) (bool, error) {
	result, err := f.contract.EvaluateTransaction("VerifyModel", id, hash)
	if err != nil {
		return false, err
	}
	return string(result) == "true", nil
}
