// cmd/cli/main.go — assemblage final
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"myr/adapters/in/cli"
	fabricadapter "myr/adapters/out/fabric"
	"myr/adapters/out/localstorage"
	"myr/domain/channel"
	"myr/domain/identity"
	"myr/domain/model"
	"myr/domain/network"
	"myr/domain/role"
)

// noopBC satisfait model.BlockchainPort pour le CLI, qui ne soumet pas d'assets.
type noopBC struct{}

var errNoFabric = fmt.Errorf("aucune connexion Fabric configurée pour le CLI")

func (noopBC) StoreModelRecord(*model.Model3D) error                { return errNoFabric }
func (noopBC) GetModelRecord(string, string) (*model.Model3D, error) { return nil, errNoFabric }
func (noopBC) ListModelRecords(string) ([]*model.Model3D, error)    { return nil, errNoFabric }
func (noopBC) VerifyIntegrity(string, string, string) (bool, error) { return false, errNoFabric }

// dataDir retourne le répertoire de données partagé avec myr-app.
// Priorité : MYR_DATA_DIR > ~/.Myr/server-data
func dataDir() string {
	if d := os.Getenv("MYR_DATA_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".Myr", "server-data")
}

func main() {
	data := dataDir()
	_ = os.MkdirAll(data, 0700)

	bc := noopBC{}
	fs := localstorage.NewLocalStorage(filepath.Join(data, "models"))

	modelSvc := model.NewService(bc, fs)

	networkStore := localstorage.NewJSONNetworkStore(filepath.Join(data, "networks.json"))
	networkSvc := network.NewService(networkStore, fabricadapter.GatewayTester{}).
		WithProvisioner(fabricadapter.NewFabricPeerProvisioner()).
		WithBootstrapper(fabricadapter.NewFabricNetworkBootstrapper())

	nodeStore := localstorage.NewJSONNodeStore(filepath.Join(data, "nodes.json"))
	channelStore := localstorage.NewJSONChannelStore(filepath.Join(data, "channels.json"))
	channelSvc := channel.NewService(channelStore).
		WithFabricConfig(fabricadapter.NewFabricChannelConfig(networkSvc, nodeStore))

	roleStore := localstorage.NewJSONRoleStore(filepath.Join(data, "roles.json"))
	roleSvc := role.NewService(roleStore)

	var caPort identity.CAPort
	if active, err := networkSvc.GetActive(); err == nil && active != nil && active.CAEndpoint != "" {
		cfg := fabricadapter.ConfigFromProfile(active)
		if ca, err := fabricadapter.NewCAClient(cfg); err == nil {
			caPort = ca
		}
	}
	walletDir := filepath.Join(data, "wallets")
	_ = os.MkdirAll(walletDir, 0700)
	identitySvc := identity.NewService(walletDir, caPort)

	cli.Execute(modelSvc, channelSvc, nil, networkSvc, roleSvc, identitySvc)
}
