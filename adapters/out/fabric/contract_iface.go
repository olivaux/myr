// adapters/out/fabric/contract_iface.go
// Interfaces d'abstraction du contrat chaincode Fabric.
// Elles permettent de substituer *client.Contract par un stub en test unitaire
// sans nécessiter de connexion gRPC réelle.
package fabric

// ContractCaller abstrait le sous-ensemble de *client.Contract utilisé par ce package :
// soumettre une transaction (écriture) et évaluer une requête (lecture seule).
type ContractCaller interface {
	SubmitTransaction(name string, args ...string) ([]byte, error)
	EvaluateTransaction(name string, args ...string) ([]byte, error)
}

// GatewayProvider abstrait *GatewayClient pour l'injection de dépendance dans FabricBlockchain.
// Il retourne un ContractCaller sur le canal par défaut ou sur un canal nommé.
type GatewayProvider interface {
	Contract() ContractCaller
	ContractFor(channelName string) ContractCaller
}
