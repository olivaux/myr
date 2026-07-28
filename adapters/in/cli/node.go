// adapters/in/cli/node.go — commandes de gestion des nœuds du réseau blockchain (UCADM03/04)
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"myr-core/domain/channel"
	"myr-core/domain/network"
)

// ── myr node ──────────────────────────────────────────────────────────────────

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage nodes in the blockchain network",
	Long: `Commands to add, provision and remove nodes of the active blockchain network.

Adding a node requires administrator privileges and that the endorsement
policy is satisfied. Removal enforces the minimum threshold of 3 active nodes (RM27).`,
}

// ── node add ──────────────────────────────────────────────────────────────────

var (
	nodeAddType    string
	nodeAddAddr    string
	nodeAddOrg     string
	nodeAddChannel string
	nodeAddCert    string
)

func runNodeAdd(w io.Writer, chSvc channel.ChannelService, netSvc network.NetworkService) error {
	channelID, err := resolveChannel(nodeAddChannel, netSvc)
	if err != nil {
		return err
	}

	nodeType := channel.NodeType(nodeAddType)
	if nodeType != channel.NodeTypePeer && nodeType != channel.NodeTypeOrderer {
		return fmt.Errorf("type de nœud invalide %q — utilisez peer ou orderer", nodeAddType)
	}

	var certs channel.NodeCerts
	if nodeAddCert != "" {
		data, err := os.ReadFile(nodeAddCert)
		if err != nil {
			return fmt.Errorf("certificat TLS introuvable : %s", nodeAddCert)
		}
		certs.TLSCert = string(data)
	}

	fmt.Fprintf(w, "Soumission de la demande d'ajout du nœud %s %s au canal %s...\n", nodeAddType, nodeAddAddr, channelID)

	if err := chSvc.AddNode(channelID, nodeType, nodeAddAddr, nodeAddOrg, certs); err != nil {
		if errors.Is(err, channel.ErrSyncTimeout) {
			fmt.Fprintf(w, "Avertissement : le nœud est ajouté mais la synchronisation a dépassé le délai. Vérifiez la connectivité avec : myr network test\n")
			return nil
		}
		if errors.Is(err, channel.ErrNodeUnreachable) {
			return fmt.Errorf("impossible de joindre %s — vérifiez le pare-feu et la disponibilité réseau", nodeAddAddr)
		}
		if errors.Is(err, channel.ErrFabricUnavailable) {
			return fmt.Errorf("adaptateur blockchain non configuré")
		}
		if errors.Is(err, channel.ErrEndorsementPolicy) {
			return fmt.Errorf("politique d'endorsement non satisfaite. Contactez les autres administrateurs")
		}
		return err
	}

	fmt.Fprintf(w, "Nœud %s %s ajouté. Synchronisation du ledger en cours.\n", nodeAddType, nodeAddAddr)
	return nil
}

var nodeAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a node to an existing network",
	Long: `Register a node in the active channel configuration.
Validates the host:port format of the address before submission.

Examples:
  myr node add --type peer --addr 203.0.113.10:7051 --org-id Org2MSP
  myr node add --type orderer --addr orderer.example.com:7050 --org-id OrdererMSP --cert tls.pem`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNodeAdd(cmd.OutOrStdout(), channelSvc, networkSvc)
	},
}

// ── node remove ───────────────────────────────────────────────────────────────

var (
	nodeRemoveAddr    string
	nodeRemoveChannel string
)

func runNodeRemove(w io.Writer, chSvc channel.ChannelService, netSvc network.NetworkService) error {
	channelID, err := resolveChannel(nodeRemoveChannel, netSvc)
	if err != nil {
		return err
	}

	if err := chSvc.RemoveNode(channelID, nodeRemoveAddr); err != nil {
		if errors.Is(err, channel.ErrNodeNotMember) {
			return fmt.Errorf("%s n'est pas membre actif du canal %s", nodeRemoveAddr, channelID)
		}
		if errors.Is(err, channel.ErrMinNodesRequired) {
			return fmt.Errorf("le réseau doit conserver au moins 3 nœuds actifs. Retrait impossible (actuellement 3 nœuds actifs). [RM27]")
		}
		if errors.Is(err, channel.ErrFabricUnavailable) {
			return fmt.Errorf("adaptateur blockchain non configuré")
		}
		if errors.Is(err, channel.ErrEndorsementPolicy) {
			return fmt.Errorf("politique d'endorsement non satisfaite. Contactez les autres administrateurs")
		}
		return err
	}

	fmt.Fprintf(w, "Nœud %s retiré du canal %s.\n", nodeRemoveAddr, channelID)
	return nil
}

var nodeRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a node from the network",
	Long: `Remove a node from the active channel configuration.
Enforces the minimum threshold of 3 active nodes (RM27) before proceeding.

Examples:
  myr node remove --addr 203.0.113.10:7051
  myr node remove --addr 203.0.113.10:7051 --channel sandbox`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNodeRemove(cmd.OutOrStdout(), channelSvc, networkSvc)
	},
}

// ── node provision ────────────────────────────────────────────────────────────

var (
	nodeProvisionID       string
	nodeProvisionHostname string
	nodeProvisionSecret   string
	nodeProvisionOutDir   string
	nodeProvisionNetwork  string
)

var nodeProvisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Register a new node's identity and generate its credentials",
	Long: `Register a node with the certificate authority of the active network
and write the cryptographic materials it needs to start into the output
directory. The exact file layout and startup instructions depend on the
blockchain adapter currently active.

Examples:
  myr node provision --node-id node1.org1.example.com --out ./node1-crypto
  myr node provision --node-id node1.org1.example.com --hostname 192.168.1.10 --out ./node1-crypto`,
	RunE: runNodeProvision,
}

func runNodeProvision(cmd *cobra.Command, args []string) error {
	if networkSvc == nil {
		return fmt.Errorf("service réseau non disponible — relancez myr avec une configuration réseau")
	}

	outDir := nodeProvisionOutDir
	if outDir == "" {
		outDir = nodeProvisionID
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Enregistrement du nœud %q auprès de l'autorité de certification...\n", nodeProvisionID)

	creds, err := networkSvc.AddPeer(nodeProvisionNetwork, network.AddPeerRequest{
		PeerID:   nodeProvisionID,
		Hostname: nodeProvisionHostname,
		Secret:   nodeProvisionSecret,
	})
	if err != nil {
		return fmt.Errorf("échec : %w", err)
	}

	fmt.Fprintln(w, "Enregistrement réussi. Génération des fichiers...")

	if err := writeNodeCredentialFiles(outDir, creds); err != nil {
		return fmt.Errorf("écriture fichiers : %w", err)
	}

	printNodeProvisionSummary(w, outDir, creds)
	return nil
}

// writeNodeCredentialFiles écrit sur disque les fichiers décrits par
// creds.Files, sans connaissance de leur mise en page (décidée par l'adapter actif).
func writeNodeCredentialFiles(outDir string, creds *network.PeerCredentials) error {
	dirs := map[string]struct{}{}
	for rel := range creds.Files {
		dirs[filepath.Dir(filepath.Join(outDir, rel))] = struct{}{}
	}
	for d := range dirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return fmt.Errorf("mkdir %s : %w", d, err)
		}
	}
	for rel, content := range creds.Files {
		path := filepath.Join(outDir, rel)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return fmt.Errorf("écriture %s : %w", path, err)
		}
	}
	return nil
}

// printNodeProvisionSummary affiche un récapitulatif générique et les
// instructions de démarrage fournies par l'adapter actif.
func printNodeProvisionSummary(w io.Writer, outDir string, creds *network.PeerCredentials) {
	abs, _ := filepath.Abs(outDir)
	sep := strings.Repeat("─", 60)

	fmt.Fprintf(w, "\n%s\n", sep)
	fmt.Fprintf(w, "Nœud %q provisionné avec succès\n", creds.PeerID)
	fmt.Fprintf(w, "%s\n\n", sep)
	fmt.Fprintf(w, "Fichiers générés dans : %s\n\n", abs)
	for rel := range creds.Files {
		fmt.Fprintf(w, "  %s\n", rel)
	}
	if creds.StartupInstructions != "" {
		fmt.Fprintf(w, "\n%s\n", sep)
		fmt.Fprintln(w, strings.ReplaceAll(creds.StartupInstructions, "<out>", abs))
	}
	fmt.Fprintf(w, "%s\n", sep)
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	nodeCmd.AddCommand(nodeAddCmd, nodeRemoveCmd, nodeProvisionCmd)

	nodeAddCmd.Flags().StringVar(&nodeAddType, "type", "", "Type de nœud : peer ou orderer (requis)")
	nodeAddCmd.Flags().StringVar(&nodeAddAddr, "addr", "", "Adresse du nœud host:port (requis)")
	nodeAddCmd.Flags().StringVar(&nodeAddOrg, "org-id", "", "Identifiant de l'organisation propriétaire du nœud (requis)")
	nodeAddCmd.Flags().StringVar(&nodeAddChannel, "channel", "", "Canal cible (défaut: canal du réseau actif)")
	nodeAddCmd.Flags().StringVar(&nodeAddCert, "cert", "", "Chemin vers le certificat TLS du nœud PEM")
	_ = nodeAddCmd.MarkFlagRequired("type")
	_ = nodeAddCmd.MarkFlagRequired("addr")
	_ = nodeAddCmd.MarkFlagRequired("org-id")

	nodeRemoveCmd.Flags().StringVar(&nodeRemoveAddr, "addr", "", "Adresse du nœud à retirer host:port (requis)")
	nodeRemoveCmd.Flags().StringVar(&nodeRemoveChannel, "channel", "", "Canal cible (défaut: canal du réseau actif)")
	_ = nodeRemoveCmd.MarkFlagRequired("addr")

	nodeProvisionCmd.Flags().StringVarP(&nodeProvisionID, "node-id", "i", "", "Identité du nœud, ex: node1.org1.example.com (requis)")
	nodeProvisionCmd.Flags().StringVar(&nodeProvisionHostname, "hostname", "", "Hostname/IP pour le SAN TLS (défaut: node-id)")
	nodeProvisionCmd.Flags().StringVar(&nodeProvisionSecret, "secret", "", "Secret d'enrôlement (généré automatiquement si omis)")
	nodeProvisionCmd.Flags().StringVarP(&nodeProvisionOutDir, "out", "o", "", "Répertoire de sortie pour les identifiants (défaut: ./<node-id>)")
	nodeProvisionCmd.Flags().StringVarP(&nodeProvisionNetwork, "network", "n", "", "ID du réseau (défaut: réseau actif)")
	_ = nodeProvisionCmd.MarkFlagRequired("node-id")
}
