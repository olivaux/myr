// chaincode/model/contract.go
package model

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

// SmartContract implémente le chaincode myrcc — périmètre minimal couvrant
// exactement les 4 fonctions attendues par domain/model/ports.go BlockchainPort
// (StoreModelRecord/GetModelRecord/ListModelRecords/VerifyIntegrity). Les noms
// de fonction et le nombre d'arguments reproduisent exactement ce qu'invoque
// adapters/out/fabric/blockchain.go côté client — ce fichier-là fait foi, pas
// les signatures indicatives de specs/3-Conception/Chaincode.md §4.
type SmartContract struct{}

func (s *SmartContract) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (s *SmartContract) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()
	switch fn {
	case "StoreModel":
		return s.storeModel(stub, args)
	case "GetModel":
		return s.getModel(stub, args)
	case "ListModels":
		return s.listModels(stub, args)
	case "VerifyModel":
		return s.verifyModel(stub, args)
	default:
		return shim.Error(fmt.Sprintf("fonction chaincode inconnue : %s", fn))
	}
}

// storeModel enregistre le Model3D reçu tel quel (aucune reformulation du JSON
// soumis), indexé par son ID — voir FabricBlockchain.StoreModelRecord.
func (s *SmartContract) storeModel(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("StoreModel attend 1 argument (model3dJSON)")
	}
	var m Model3D
	if err := json.Unmarshal([]byte(args[0]), &m); err != nil {
		return shim.Error(fmt.Sprintf("StoreModel : JSON invalide : %v", err))
	}
	if m.ID == "" {
		return shim.Error("StoreModel : ID vide")
	}
	if err := stub.PutState(m.ID, []byte(args[0])); err != nil {
		return shim.Error(fmt.Sprintf("StoreModel : écriture ledger : %v", err))
	}
	return shim.Success([]byte(stub.GetTxID()))
}

// getModel retourne le JSON stocké tel quel — FabricBlockchain.GetModelRecord
// le désérialise directement dans *model.Model3D côté client.
func (s *SmartContract) getModel(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("GetModel attend 1 argument (id)")
	}
	data, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(fmt.Sprintf("GetModel : lecture ledger : %v", err))
	}
	if data == nil {
		return shim.Error(fmt.Sprintf("GetModel : asset %s introuvable", args[0]))
	}
	return shim.Success(data)
}

// listModels retourne un tableau JSON de tous les Model3D du ledger, filtré par
// ChannelID si non vide. FabricBlockchain.ListModelRecords désérialise
// directement la réponse dans []*model.Model3D.
func (s *SmartContract) listModels(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	channelID := ""
	if len(args) > 0 {
		channelID = args[0]
	}
	iter, err := stub.GetStateByRange("", "")
	if err != nil {
		return shim.Error(fmt.Sprintf("ListModels : itération ledger : %v", err))
	}
	defer iter.Close()

	results := make([]json.RawMessage, 0)
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("ListModels : lecture ledger : %v", err))
		}
		if channelID != "" {
			var m Model3D
			if jsonErr := json.Unmarshal(kv.Value, &m); jsonErr != nil {
				continue // enregistrement non conforme — ignoré plutôt que de faire échouer tout le listing
			}
			if m.ChannelID != channelID {
				continue
			}
		}
		results = append(results, json.RawMessage(kv.Value))
	}
	out, err := json.Marshal(results)
	if err != nil {
		return shim.Error(fmt.Sprintf("ListModels : sérialisation : %v", err))
	}
	return shim.Success(out)
}

// verifyModel compare le hash stocké au hash fourni — FabricBlockchain.VerifyIntegrity
// attend "true"/"false" en texte brut, pas du JSON.
func (s *SmartContract) verifyModel(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
		return shim.Error("VerifyModel attend 2 arguments (id, hash)")
	}
	data, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(fmt.Sprintf("VerifyModel : lecture ledger : %v", err))
	}
	if data == nil {
		return shim.Success([]byte("false"))
	}
	var m Model3D
	if err := json.Unmarshal(data, &m); err != nil {
		return shim.Error(fmt.Sprintf("VerifyModel : JSON invalide en ledger : %v", err))
	}
	if m.Hash == args[1] {
		return shim.Success([]byte("true"))
	}
	return shim.Success([]byte("false"))
}
