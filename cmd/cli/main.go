// cmd/cli/main.go — assemblage final
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"myr-core/adapters/in/cli"
	fabricadapter "myr-core/adapters/out/fabric"
	"myr-core/adapters/out/localstorage"
	"myr-core/adapters/out/webimage"
	"myr-core/domain/channel"
	"myr-core/domain/identity"
	"myr-core/domain/model"
	"myr-core/domain/network"
	"myr-core/domain/payment"
	"myr-core/domain/role"
	"myr-core/domain/session"
)

// noopPaymentGateway satisfait payment.PaymentGatewayPort tant qu'aucun adapter
// de paiement n'est câblé — évite un service nil (panique) côté CLI.
type noopPaymentGateway struct{}

var errNoPaymentGateway = fmt.Errorf("aucun gateway de paiement configuré pour le CLI")

// version est injectée via -ldflags "-X main.version=..." lors de la compilation
// (voir Makefile : VERSION dérivé de `git describe --tags`).
var version = "dev"

func (noopPaymentGateway) Transfer(*payment.Payment) error { return errNoPaymentGateway }
func (noopPaymentGateway) GetHistory(string) ([]*payment.Payment, error) {
	return nil, errNoPaymentGateway
}

// dataDir retourne le répertoire de données partagé avec myr-api.
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

	networkStore := localstorage.NewJSONNetworkStore(filepath.Join(data, "networks.json"))
	networkSvc := network.NewService(networkStore, fabricadapter.GatewayTester{}).
		WithProvisioner(fabricadapter.NewFabricPeerProvisioner()).
		WithBootstrapper(fabricadapter.NewFabricNetworkBootstrapper())

	active, _ := networkSvc.GetActive()

	// ── Connexion Gateway Fabric (non bloquante) ──────────────────────────────
	// bc reste nil si Fabric est indisponible — pas de repli silencieux vers un
	// autre stockage : les opérations sur les modèles retournent alors
	// model.ErrBlockchainUnavailable (voir domain/model/service.go).
	var bc model.BlockchainPort
	if active != nil {
		cfg := fabricadapter.ConfigFromProfile(active)
		if gw, err := fabricadapter.NewGatewayClient(cfg); err == nil {
			defer gw.Close()
			bc = fabricadapter.NewFabricBlockchain(gw)
		}
	}
	fs := localstorage.NewLocalStorage(filepath.Join(data, "models"))
	// store partage le même fichier assets.json que myr-api (voir dataDir ci-dessus) —
	// donne au CLI accès aux connexions/miniatures/interfaces hors blockchain gérées côté GUI.
	store := localstorage.NewJSONBlockchain(filepath.Join(data, "assets.json"))
	modelSvc := model.NewService(bc, fs).
		WithConnStore(store).
		WithThumbStore(store).
		WithIfaceStore(store).
		WithOGImageFetcher(webimage.New())

	nodeStore := localstorage.NewJSONNodeStore(filepath.Join(data, "nodes.json"))
	channelStore := localstorage.NewJSONChannelStore(filepath.Join(data, "channels.json"))
	channelSvc := channel.NewService(channelStore).
		WithFabricConfig(fabricadapter.NewFabricChannelConfig(networkSvc, nodeStore))

	roleStore := localstorage.NewJSONRoleStore(filepath.Join(data, "roles.json"))
	roleSvc := role.NewService(roleStore)

	var caPort identity.CAPort
	if active != nil && active.CAEndpoint != "" {
		cfg := fabricadapter.ConfigFromProfile(active)
		if ca, err := fabricadapter.NewCAClient(cfg); err == nil {
			caPort = ca
		}
	}
	walletDir := filepath.Join(data, "wallets")
	_ = os.MkdirAll(walletDir, 0700)
	identitySvc := identity.NewService(walletDir, caPort)

	paymentSvc := payment.NewService(noopPaymentGateway{})

	sessionStore := localstorage.NewJSONSessionStore(filepath.Join(data, "session.json"))
	sessionSvc := session.NewService(sessionStore)

	cli.Version = version
	cli.Execute(modelSvc, channelSvc, paymentSvc, networkSvc, roleSvc, identitySvc, sessionSvc)
}
