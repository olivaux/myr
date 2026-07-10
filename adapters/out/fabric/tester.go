// adapters/out/fabric/tester.go
package fabric

import (
	"myr/domain/network"
)

// GatewayTester implémente network.ConnectionTester via une connexion Gateway complète.
// Il tente le handshake gRPC + TLS + identité X.509 pour valider la config réseau.
type GatewayTester struct{}

func (GatewayTester) Test(n *network.NetworkProfile) error {
	// Vérifier que le profil réseau a tous les champs requis
	if err := ValidateNetworkProfile(n); err != nil {
		return err
	}

	// Convertir le profil en Config Fabric
	cfg := ConfigFromNetworkProfile(n)

	// Tenter une connexion gateway
	gw, err := NewGatewayClient(cfg)
	if err != nil {
		return err
	}
	gw.Close()
	return nil
}
