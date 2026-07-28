// adapters/out/fabric/config.go
package fabric

import (
	"os"

	"myr-core/domain/network"
)

// Config contient les paramètres de connexion au réseau Hyperledger Fabric.
type Config struct {
	// Adresse gRPC du peer (ex: "localhost:XXXX")
	PeerEndpoint string
	// Nom TLS du peer (ex: "peer0.org1.example.com")
	GatewayPeer string
	// MSP ID de l'organisation cliente (ex: "Org1MSP")
	MSPID string
	// Chemin vers le certificat client (PEM)
	CertPath string
	// Chemin vers la clé privée du client (PEM)
	KeyPath string
	// Chemin vers le certificat TLS du peer (PEM)
	TLSCertPath string
	// Nom du channel Fabric (ex: "mychannel")
	FabricChannel string
	// Nom du chaincode déployé (ex: "myrcc")
	ChaincodeName string
	// Adresse HTTPS du Fabric CA (ex: "https://XXX.XXX.XXX.XXX:YYYY") — optionnel
	CAEndpoint string
	// Nom du CA (ex: "ca-org1") — optionnel
	CAName string
	// Credentials admin CA — distincts du cert peer.
	// Si vides, NewCAClient utilise CertPath/KeyPath comme fallback.
	// Sur le serveur MYR, ces champs pointent vers le compte admin de la CA Fabric.
	CAAdminCertPath string
	CAAdminKeyPath  string
}

// ConfigFromEnv construit une Config à partir des variables d'environnement.
// Valeurs par défaut compatibles avec fabric-samples/test-network.
func ConfigFromEnv() Config {
	return Config{
		PeerEndpoint:    getenv("FABRIC_PEER_ENDPOINT", "localhost:7051"),
		GatewayPeer:     getenv("FABRIC_GATEWAY_PEER", "peer0.org1.example.com"),
		MSPID:           getenv("FABRIC_MSP_ID", "Org1MSP"),
		CertPath:        getenv("FABRIC_CERT_PATH", ""),
		KeyPath:         getenv("FABRIC_KEY_PATH", ""),
		TLSCertPath:     getenv("FABRIC_TLS_CERT_PATH", ""),
		FabricChannel:   getenv("FABRIC_CHANNEL_NAME", "mychannel"),
		ChaincodeName:   getenv("FABRIC_CHAINCODE_NAME", "myrcc"),
		CAEndpoint:      getenv("FABRIC_CA_ENDPOINT", ""),
		CAName:          getenv("FABRIC_CA_NAME", ""),
		CAAdminCertPath: getenv("FABRIC_CA_ADMIN_CERT_PATH", ""),
		CAAdminKeyPath:  getenv("FABRIC_CA_ADMIN_KEY_PATH", ""),
	}
}

// ConfigFromProfile convertit un domain.NetworkProfile en Config Fabric.
// Utilisé par NetworkPool.LoadFromProfiles pour ouvrir les connexions au démarrage.
func ConfigFromProfile(np *network.NetworkProfile) Config {
	return Config{
		PeerEndpoint:    np.PeerEndpoint,
		GatewayPeer:     np.GatewayPeer,
		MSPID:           np.MSPID,
		CertPath:        np.CertPath,
		KeyPath:         np.KeyPath,
		TLSCertPath:     np.TLSCertPath,
		FabricChannel:   np.FabricChannel,
		ChaincodeName:   np.GetChaincodeName(),
		CAEndpoint:      np.CAEndpoint,
		CAName:          np.CAName,
		CAAdminCertPath: np.CAAdminCertPath,
		CAAdminKeyPath:  np.CAAdminKeyPath,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
