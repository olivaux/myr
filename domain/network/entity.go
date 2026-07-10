// domain/network/entity.go
package network

import "time"

// ChaincodeConfig décrit le chaincode déployé sur ce réseau.
// C'est réseau-spécifique : chaque entreprise peut avoir son propre chaincode.
type ChaincodeConfig struct {
	// Nom du chaincode (ex: "myrcc")
	Name string `json:"name"`
	// Source/URL du chaincode compilé (ex: "https://registry.acme.com/myrcc@v1.2.3")
	// Vide si le chaincode est déjà déployé et accessible directement
	Source string `json:"source,omitempty"`
	// Version du chaincode (ex: "1.2.3")
	Version string `json:"version,omitempty"`
	// Checksum SHA256 pour vérifier l'intégrité du binaire téléchargé
	Checksum string `json:"checksum,omitempty"`
}

// NetworkProfile représente un réseau Hyperledger Fabric configuré dans myr.
type NetworkProfile struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	PeerEndpoint  string          `json:"peer_endpoint"`       // ex: "localhost:7051"
	GatewayPeer   string          `json:"gateway_peer"`        // nom TLS du peer, ex: "peer0.org1.example.com"
	MSPID         string          `json:"msp_id"`              // ex: "Org1MSP"
	CertPath      string          `json:"cert_path"`           // chemin vers le certificat client (PEM)
	KeyPath       string          `json:"key_path"`            // chemin vers la clé privée (PEM)
	TLSCertPath   string          `json:"tls_cert_path"`       // chemin vers le certificat TLS du peer
	FabricChannel string          `json:"fabric_channel"`      // canal par defaut pour la gateway
	Channels      []string        `json:"channels"`            // canaux Fabric disponibles (depuis le profil de connexion)
	ChaincodeName string          `json:"chaincode_name"`      // DEPRECATED: utiliser Chaincode.Name à la place
	Chaincode     ChaincodeConfig `json:"chaincode,omitempty"` // configs du chaincode réseau-spécifique
	CAEndpoint    string          `json:"ca_endpoint"`         // ex: "https://203.0.113.10:7054"
	CAName        string          `json:"ca_name"`             // ex: "ca-org1"
	// OrdererEndpoint est l'adresse de l'orderer utilisée pour soumettre les mises
	// à jour de configuration de canal (myr org add / node add / node remove).
	// Renseigné automatiquement par `myr network create` ; vide pour un profil
	// importé ou ajouté manuellement sans information d'orderer.
	OrdererEndpoint string `json:"orderer_endpoint,omitempty"`
	// OrdererTLSCACertPath pointe vers le certificat racine TLS à utiliser pour
	// joindre l'orderer ci-dessus. Vide = repli sur TLSCertPath.
	OrdererTLSCACertPath string `json:"orderer_tls_ca_cert_path,omitempty"`
	// CAAdminCertPath/CAAdminKeyPath pointent vers une identité disposant des
	// privilèges de registrar Fabric CA (hf.Registrar.*) — nécessaires pour
	// Register/UpdateAttributes (myr identity set-role, auto-register...).
	// Distincts de CertPath/KeyPath (identité applicative de l'org, sans ces
	// privilèges) : Register avec CertPath/KeyPath échoue par autorisation.
	// Vides = repli sur CertPath/KeyPath (adapters/out/fabric.NewCAClient).
	CAAdminCertPath string `json:"ca_admin_cert_path,omitempty"`
	CAAdminKeyPath  string `json:"ca_admin_key_path,omitempty"`
	// ServerURL est l'URL du MYR Server central de l'organisation.
	// Quand ce champ est renseigné, le GUI client y redirige les demandes d'identité
	// (POST /api/identity/request, POST /api/identity/guest) au lieu de les traiter
	// localement. Le MYR Server détient les credentials admin CA et gère les comptes.
	// Exemple : "https://myr.monorg.com:8080"
	// Vide = le GUI gère lui-même (mode standalone, dev uniquement).
	ServerURL string `json:"server_url,omitempty"`
	// AllowAutoGuest contrôle si un inconnu peut obtenir automatiquement un accès
	// en lecture seule sans validation préalable de l'admin.
	// false (défaut) : tout accès requiert une identité validée par l'admin.
	// true  : POST /api/identity/guest délivre immédiatement un token reader.
	AllowAutoGuest bool `json:"allow_auto_guest,omitempty"`
	// AllowAutoRegister contrôle si une demande de compte est automatiquement
	// enregistrée dans la CA Fabric (requiert des credentials admin configurés).
	// false (défaut) : la demande reste en attente, l'admin crée le compte manuellement.
	// true  : POST /api/identity/request enregistre l'identité dans la CA et retourne
	//         le secret d'enrollment directement dans la réponse.
	AllowAutoRegister bool `json:"allow_auto_register,omitempty"`
	// AutoRegisterRole définit le rôle attribué automatiquement lors d'un auto-register.
	// Valeurs possibles : "reader" | "contributor" | "auditor"
	// Vide = "reader" par défaut.
	// Exemple pour un réseau ouvert aux concepteurs : "contributor"
	AutoRegisterRole string    `json:"auto_register_role,omitempty"`
	Active           bool      `json:"active"`
	IsProduction     bool      `json:"is_production,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// GetChaincodeName retourne le nom du chaincode avec retrocompatibilité.
func (n *NetworkProfile) GetChaincodeName() string {
	if n.Chaincode.Name != "" {
		return n.Chaincode.Name
	}
	return n.ChaincodeName // fallback ancien format
}
