// adapters/out/fabric/config_builder.go
// Convertit un NetworkProfile en Config Fabric

package fabric

import (
	"fmt"

	"myr-core/domain/network"
)

// ConfigFromNetworkProfile construit une Config Fabric depuis un NetworkProfile.
// Utilise les métadonnées du chaincode incluses dans le profil.
func ConfigFromNetworkProfile(np *network.NetworkProfile) Config {
	chaincodeName := np.GetChaincodeName()
	return Config{
		PeerEndpoint:  np.PeerEndpoint,
		GatewayPeer:   np.GatewayPeer,
		MSPID:         np.MSPID,
		CertPath:      np.CertPath,
		KeyPath:       np.KeyPath,
		TLSCertPath:   np.TLSCertPath,
		FabricChannel: np.FabricChannel,
		ChaincodeName: chaincodeName,
		CAEndpoint:    np.CAEndpoint,
		CAName:        np.CAName,
	}
}

// ValidateNetworkProfile vérifie que le NetworkProfile a tous les champs requis pour Fabric.
func ValidateNetworkProfile(np *network.NetworkProfile) error {
	if np.PeerEndpoint == "" {
		return fmt.Errorf("peer_endpoint manquant dans le profil réseau")
	}
	if np.GatewayPeer == "" {
		return fmt.Errorf("gateway_peer manquant dans le profil réseau")
	}
	if np.MSPID == "" {
		return fmt.Errorf("msp_id manquant dans le profil réseau")
	}
	if np.TLSCertPath == "" {
		return fmt.Errorf("tls_cert_path manquant dans le profil réseau")
	}
	if np.FabricChannel == "" {
		return fmt.Errorf("fabric_channel manquant dans le profil réseau")
	}
	if np.GetChaincodeName() == "" {
		return fmt.Errorf("chaincode.name manquant (ou chaincode_name legacy)")
	}
	return nil
}
