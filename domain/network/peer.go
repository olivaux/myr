// domain/network/peer.go — entités pour l'ajout d'un nœud au réseau blockchain.
package network

// AddPeerRequest regroupe les paramètres pour l'enregistrement d'un nouveau nœud.
type AddPeerRequest struct {
	// PeerID est l'identité CA du nœud, ex: "node1.org1.example.com".
	PeerID string
	// Hostname est le FQDN ou l'IP utilisé pour le SAN du certificat TLS.
	// Défaut : PeerID.
	Hostname string
	// Secret est le mot de passe d'enrôlement.
	// Laissez vide pour que la CA en génère un automatiquement.
	Secret string
}

// PeerCredentials contient les matériaux cryptographiques générés pour démarrer un nœud.
type PeerCredentials struct {
	PeerID string
	// Certificats de signature
	SignCert string
	SignKey  string
	CACert   string
	// Certificats TLS
	TLSCert string
	TLSKey  string
	// Files associe un chemin relatif à écrire sur disque à son contenu.
	// La mise en page (noms de dossiers/fichiers) est décidée par l'adapter
	// actif — le CLI ne fait que les écrire tels quels.
	Files map[string]string
	// StartupInstructions est un texte d'aide, propre à la technologie
	// active, expliquant comment démarrer le nœud avec ces fichiers.
	StartupInstructions string
}

// CreateNetworkRequest regroupe les paramètres nécessaires pour créer un réseau
// blockchain from scratch (UCADM02 flux nominal), sur la machine où myr s'exécute.
type CreateNetworkRequest struct {
	Name string // nom lisible du réseau, ex: "diy-network"
	// OrgMSPID est l'identifiant de la première organisation du réseau, ex: "Org1MSP".
	OrgMSPID string
	// OrgName est le nom lisible de cette organisation.
	OrgName string
	// Domain est le domaine DNS logique utilisé pour nommer les nœuds
	// (ex: "diy-network.com" → peer0.org1.diy-network.com).
	Domain string
	// ChannelName est le nom du canal applicatif créé avec le réseau.
	ChannelName string
	// NumPeers est le nombre de peers de l'organisation initiale (défaut : 2).
	// Avec l'orderer, cela porte le réseau à NumPeers+1 nœuds (3 par défaut).
	NumPeers int
	// CommissionRate est le taux de commission (%) appliqué aux livraisons du réseau (RM29).
	CommissionRate float64
	// Currency est la devise unique du réseau (RM33), ex: "EUR".
	Currency string
}

// BootstrapResult décrit l'issue du bootstrap d'un réseau from scratch.
type BootstrapResult struct {
	Profile *NetworkProfile
	// Nodes liste les nœuds provisionnés (adresse + type), pour bookkeeping côté adapter appelant.
	Nodes []BootstrapNode
}

// BootstrapNode décrit un nœud provisionné lors du bootstrap.
type BootstrapNode struct {
	Type string // "peer" | "orderer"
	Addr string // host:port
}
