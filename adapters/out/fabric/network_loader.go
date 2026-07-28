// adapters/out/fabric/network_loader.go
// Charge les profils réseau depuis config/networks/*.yaml

package fabric

import (
	"fmt"
	"os"
	"path/filepath"

	"myr-core/domain/network"

	"gopkg.in/yaml.v3"
)

// LoadNetworkProfile charge un profil réseau depuis un fichier YAML.
// Exemple : LoadNetworkProfile("config/networks", "org1-fabric")
// cherche config/networks/org1-fabric.yaml
func LoadNetworkProfile(configDir, networkID string) (*network.NetworkProfile, error) {
	filePath := filepath.Join(configDir, "networks", networkID+".yaml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("impossible de lire %s : %w", filePath, err)
	}

	var np network.NetworkProfile
	if err := yaml.Unmarshal(data, &np); err != nil {
		return nil, fmt.Errorf("erreur parsing YAML %s : %w", filePath, err)
	}

	// Validation minimale
	if np.ID == "" {
		return nil, fmt.Errorf("%s : champ 'id' requis", filePath)
	}
	if np.Name == "" {
		return nil, fmt.Errorf("%s : champ 'name' requis", filePath)
	}

	return &np, nil
}

// LoadNetworkProfileFromEnv charge le profil réseau indiqué par la variable FABRIC_NETWORK_ID.
// Cherche le fichier dans configDir/networks/.
//
// Exemple :
//
//	export FABRIC_NETWORK_ID="org1-fabric"  # cherche config/networks/org1-fabric.yaml
//	np, err := LoadNetworkProfileFromEnv("config", os.Getenv("FABRIC_NETWORK_ID"))
func LoadNetworkProfileFromEnv(configDir, networkID string) (*network.NetworkProfile, error) {
	if networkID == "" {
		return nil, fmt.Errorf("FABRIC_NETWORK_ID non configuré")
	}
	return LoadNetworkProfile(configDir, networkID)
}
