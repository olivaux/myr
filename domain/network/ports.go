// domain/network/ports.go
package network

// Repo persiste les profils réseau.
type Repo interface {
	Save(n *NetworkProfile) error
	FindAll() ([]*NetworkProfile, error)
	Delete(id string) error
}

// ConnectionTester teste une connexion réelle vers un profil réseau.
// Injecté depuis la couche adapter pour éviter que le domaine connaisse Fabric.
type ConnectionTester interface {
	Test(n *NetworkProfile) error
}

// PeerProvisioner provisionne les matériaux cryptographiques d'un nouveau peer
// via l'API REST de la Fabric CA.
// Reçoit le profil réseau complet (CA endpoint, certs admin) et la requête.
type PeerProvisioner interface {
	RegisterAndProvision(net *NetworkProfile, req AddPeerRequest) (*PeerCredentials, error)
}

// NetworkBootstrapper crée un réseau blockchain from scratch (UCADM02 flux nominal) :
// démarrage de la CA, génération des identités et de la configuration du canal,
// démarrage des nœuds, rejoint du canal. Opère sur la machine où myr s'exécute
// (cf. principe d'exécution distante).
type NetworkBootstrapper interface {
	Bootstrap(req CreateNetworkRequest) (*BootstrapResult, error)
}
