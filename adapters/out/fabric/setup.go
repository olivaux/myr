// adapters/out/fabric/setup.go
// Fonctions utilitaires pour initialiser l'adapter Fabric depuis les profils réseau

package fabric

import (
	"fmt"
)

// SetupGatewayFromNetworkProfile ouvre une connexion gateway en chargeant
// un profil réseau depuis config/networks/{networkID}.yaml
//
// Workflow typique :
//
//	gc, err := SetupGatewayFromNetworkProfile("config", "org1-fabric")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer gc.Close()
//	// Utiliser gc (GatewayClient)
func SetupGatewayFromNetworkProfile(configDir, networkID string) (*GatewayClient, error) {
	// 1. Charger le profil réseau depuis YAML
	np, err := LoadNetworkProfile(configDir, networkID)
	if err != nil {
		return nil, fmt.Errorf("impossible de charger le profil réseau %s : %w", networkID, err)
	}

	// 2. Valider le profil
	if err := ValidateNetworkProfile(np); err != nil {
		return nil, fmt.Errorf("validation du profil réseau %s échouée : %w", networkID, err)
	}

	// 3. Convertir en Config Fabric
	cfg := ConfigFromNetworkProfile(np)

	// 4. Ouvrir la connexion gateway
	gc, err := NewGatewayClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir la gateway pour %s : %w", networkID, err)
	}

	return gc, nil
}
