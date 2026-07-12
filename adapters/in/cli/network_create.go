// adapters/in/cli/network_create.go — création d'un réseau blockchain from scratch (UCADM02)
package cli

import (
	"fmt"
	"io"

	"myr/domain/network"

	"github.com/spf13/cobra"
)

var (
	networkCreateName           string
	networkCreateOrgID          string
	networkCreateOrgName        string
	networkCreateDomain         string
	networkCreateChannel        string
	networkCreatePeers          int
	networkCreateCommissionRate float64
	networkCreateCurrency       string
	networkCreateActivate       bool
)

func runNetworkCreateCmd(w io.Writer, svc network.NetworkService, req network.CreateNetworkRequest, activate bool) error {
	fmt.Fprintf(w, "Création du réseau %q — organisation %s, canal %s, %d nœud(s)...\n", req.Name, req.OrgMSPID, req.ChannelName, req.NumPeers+1)
	fmt.Fprintln(w, "Cette opération démarre des conteneurs Docker. Cela peut prendre plusieurs minutes.")

	n, err := svc.Create(req)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Réseau %q créé (id: %s).\n", n.Name, n.ID)
	fmt.Fprintf(w, "  Endpoint  : %s\n", n.PeerEndpoint)
	fmt.Fprintf(w, "  Org ID    : %s\n", n.MSPID)
	fmt.Fprintf(w, "  Canal     : %s\n", n.FabricChannel)
	fmt.Fprintf(w, "  CA        : %s\n", n.CAEndpoint)
	fmt.Fprintln(w, "Le chaincode n'est pas déployé automatiquement — déployez-le séparément avant utilisation.")

	if activate {
		if err := svc.Activate(n.ID); err != nil {
			return fmt.Errorf("réseau créé mais échec de l'activation : %w", err)
		}
		fmt.Fprintf(w, "Réseau %q activé.\n", n.Name)
	} else {
		fmt.Fprintf(w, "Activez-le avec : myr network activate %s\n", n.ID)
	}
	return nil
}

var networkCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a brand new blockchain network from scratch",
	Long: `Bootstraps a brand new blockchain network from scratch, on the machine
where myr is running — use this when no network exists yet at all (a freshly
installed server). If a network already exists somewhere and you just need
to connect myr to it, use "myr network add" or "myr network import" instead.

This command starts a certificate authority, provisions the
organisation/orderer/peer identities, generates the channel, starts the
orderer and peer nodes (3 nodes by default: 1 orderer + 2 peers, for
resilience — a network needs at least 3 active nodes to tolerate removing
one later), and makes them join the channel.

The chaincode (smart contract) is NOT deployed automatically — deploy it
separately once the network is up.

Examples:
  myr network create --name diy-network --org-id Org1MSP --org-name "Org 1" \
    --domain diy-network.com --channel sandbox

  myr network create --name diy-network --org-id Org1MSP --org-name "Org 1" \
    --domain diy-network.com --channel sandbox --peers 3 --activate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		req := network.CreateNetworkRequest{
			Name:           networkCreateName,
			OrgMSPID:       networkCreateOrgID,
			OrgName:        networkCreateOrgName,
			Domain:         networkCreateDomain,
			ChannelName:    networkCreateChannel,
			NumPeers:       networkCreatePeers,
			CommissionRate: networkCreateCommissionRate,
			Currency:       networkCreateCurrency,
		}
		return runNetworkCreateCmd(cmd.OutOrStdout(), networkSvc, req, networkCreateActivate)
	},
}

func init() {
	networkCmd.AddCommand(networkCreateCmd)

	networkCreateCmd.Flags().StringVar(&networkCreateName, "name", "", "Nom lisible du réseau (requis)")
	networkCreateCmd.Flags().StringVar(&networkCreateOrgID, "org-id", "", "Identifiant de l'organisation initiale (requis)")
	networkCreateCmd.Flags().StringVar(&networkCreateOrgName, "org-name", "", "Nom lisible de l'organisation initiale (requis)")
	networkCreateCmd.Flags().StringVar(&networkCreateDomain, "domain", "", "Domaine logique des nœuds, ex: diy-network.com (requis)")
	networkCreateCmd.Flags().StringVar(&networkCreateChannel, "channel", "", "Nom du canal applicatif créé avec le réseau (requis)")
	networkCreateCmd.Flags().IntVar(&networkCreatePeers, "peers", 2, "Nombre de peers de l'organisation initiale (min. 2, soit 3 nœuds avec l'orderer — seuil minimum pour la résilience du réseau)")
	networkCreateCmd.Flags().Float64Var(&networkCreateCommissionRate, "commission-rate", 10, "Taux de commission (%) appliqué aux livraisons du réseau")
	networkCreateCmd.Flags().StringVar(&networkCreateCurrency, "currency", "EUR", "Devise unique du réseau")
	networkCreateCmd.Flags().BoolVar(&networkCreateActivate, "activate", false, "Activer automatiquement le réseau créé")
	_ = networkCreateCmd.MarkFlagRequired("name")
	_ = networkCreateCmd.MarkFlagRequired("org-id")
	_ = networkCreateCmd.MarkFlagRequired("org-name")
	_ = networkCreateCmd.MarkFlagRequired("domain")
	_ = networkCreateCmd.MarkFlagRequired("channel")
}
