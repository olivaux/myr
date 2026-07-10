// domain/network/port_in.go — port d'entrée : interface exposée aux adapters entrants
package network

// NetworkService est le port d'entrée du domaine network.
// Il expose les opérations de gestion des profils réseau Fabric.
type NetworkService interface {
	Add(name, peerEndpoint, gatewayPeer, mspID, certPath, keyPath, tlsCertPath, fabricChannel, chaincodeName, caEndpoint, caName string, channels []string, allowAutoGuest, allowAutoRegister bool, autoRegisterRole string, isProduction bool) (*NetworkProfile, error)
	Update(id, name, peerEndpoint, gatewayPeer, mspID, certPath, keyPath, tlsCertPath, fabricChannel, chaincodeName, caEndpoint, caName string, allowAutoGuest, allowAutoRegister bool, autoRegisterRole string, isProduction bool) (*NetworkProfile, error)
	List() ([]*NetworkProfile, error)
	GetActive() (*NetworkProfile, error)
	Activate(id string) error
	Delete(id string) error
	TestConnection(id string) error
	// AddPeer enregistre un nouveau peer dans la CA du réseau et retourne ses certificats.
	// networkID vide = réseau actif. Nécessite que le réseau ait une CA configurée.
	AddPeer(networkID string, req AddPeerRequest) (*PeerCredentials, error)
	// Create crée un réseau blockchain from scratch (UCADM02) sur la machine où myr
	// s'exécute : démarre la CA, provisionne les identités, génère la configuration
	// du canal, démarre les nœuds (3 par défaut) et enregistre le profil résultant.
	// Nécessite qu'un NetworkBootstrapper soit injecté (WithBootstrapper).
	Create(req CreateNetworkRequest) (*NetworkProfile, error)
}
