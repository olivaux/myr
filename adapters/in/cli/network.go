// adapters/in/cli/network.go — commandes de gestion des profils réseau
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"myr/domain/network"

	"github.com/spf13/cobra"
)

// ── myr network ───────────────────────────────────────────────────────────────

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Create, configure and manage blockchain networks",
	Long: `Commands to create and manage blockchain networks.

A "network profile" is the local configuration myr uses to connect to a
blockchain node (endpoint, organisation ID, certificates, certificate
authority, channel). Myr's own connection to the network is always through a
single node per organisation — the underlying blockchain synchronises the
rest between nodes.

There are three different ways to end up with a network profile, depending
on whether the underlying infrastructure already exists:

  - create : nothing exists yet — bootstraps a brand new blockchain network
             on this machine (starts the certificate authority, generates
             identities, starts the nodes) and registers the resulting
             profile. Use this to start from a blank server.
  - add    : the infrastructure already exists (a node is already running
             somewhere) — just registers a profile pointing at it. Does not
             start or configure anything.
  - import : same as add, but reads the connection details from a file
             instead of typing them as flags.

Examples:
  myr network create --name diy-network --org-id Org1MSP --org-name "Org 1" --domain diy-network.com --channel sandbox
  myr network add --name sandbox --node 203.0.113.10:7051 --org-id Org1MSP
  myr network import --profile connection-org1.json
  myr network list
  myr network activate net-1700000000000`,
}

// ── list ──────────────────────────────────────────────────────────────────────

func runNetworkList(w io.Writer, svc network.NetworkService) error {
	networks, err := svc.List()
	if err != nil {
		return err
	}
	if len(networks) == 0 {
		fmt.Fprintln(w, `Aucun réseau configuré. Si réseau existant, Utilisez "myr network add" ou "myr network import". Sinon "myr network create" pour créer depuis un serveur vierge.`)
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNOM\tENDPOINT\tORG-ID\tACTIF")
	for _, n := range networks {
		actif := ""
		if n.Active {
			actif = "*"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", n.ID, n.Name, n.PeerEndpoint, n.MSPID, actif)
	}
	return tw.Flush()
}

var networkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured network profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetworkList(cmd.OutOrStdout(), networkSvc)
	},
}

// ── show ──────────────────────────────────────────────────────────────────────

func runNetworkShow(w io.Writer, svc network.NetworkService, id string) error {
	networks, err := svc.List()
	if err != nil {
		return err
	}
	for _, n := range networks {
		if n.ID != id {
			continue
		}
		sep := strings.Repeat("─", 60)
		serverURL := n.ServerURL
		if serverURL == "" {
			serverURL = "(non configuré)"
		}
		autoRole := n.AutoRegisterRole
		if autoRole == "" {
			autoRole = "reader"
		}
		fmt.Fprintf(w, "%s\n", sep)
		fmt.Fprintf(w, "Réseau : %s  (id: %s)\n", n.Name, n.ID)
		fmt.Fprintf(w, "%s\n", sep)
		fmt.Fprintf(w, "Endpoint du nœud : %s\n", n.PeerEndpoint)
		fmt.Fprintf(w, "Nœud d'entrée TLS: %s\n", n.GatewayPeer)
		fmt.Fprintf(w, "Org ID           : %s\n", n.MSPID)
		fmt.Fprintf(w, "Canal            : %s\n", n.FabricChannel)
		fmt.Fprintf(w, "Certificat       : %s\n", n.CertPath)
		fmt.Fprintf(w, "Clé privée       : %s\n", n.KeyPath)
		fmt.Fprintf(w, "Certificat TLS   : %s\n", n.TLSCertPath)
		fmt.Fprintf(w, "Contrat          : %s\n", n.GetChaincodeName())
		fmt.Fprintf(w, "Autorité cert.   : %s\n", n.CAEndpoint)
		fmt.Fprintf(w, "Nom autorité     : %s\n", n.CAName)
		fmt.Fprintf(w, "Server URL       : %s\n", serverURL)
		fmt.Fprintf(w, "Auto guest       : %v\n", n.AllowAutoGuest)
		fmt.Fprintf(w, "Auto register    : %v\n", n.AllowAutoRegister)
		fmt.Fprintf(w, "Auto role        : %s\n", autoRole)
		fmt.Fprintf(w, "Production       : %v\n", n.IsProduction)
		fmt.Fprintf(w, "Actif            : %v\n", n.Active)
		fmt.Fprintf(w, "Créé le          : %s\n", n.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "%s\n", sep)
		return nil
	}
	return fmt.Errorf("réseau introuvable : %s", id)
}

var networkShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show details of a network profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetworkShow(cmd.OutOrStdout(), networkSvc, args[0])
	},
}

// ── add ───────────────────────────────────────────────────────────────────────

var (
	networkAddName         string
	networkAddNode         string
	networkAddOrgID        string
	networkAddGateway      string
	networkAddCert         string
	networkAddKey          string
	networkAddTLSCert      string
	networkAddChannel      string
	networkAddContract     string
	networkAddCA           string
	networkAddCAName       string
	networkAddAutoGuest    bool
	networkAddAutoRegister bool
	networkAddAutoRole     string
	networkAddProduction   bool
)

func runNetworkAdd(w io.Writer, svc network.NetworkService, name, node, gateway, orgID, cert, key, tlsCert, channel, contract, ca, caName string, autoGuest, autoRegister bool, autoRole string, production bool) error {
	n, err := svc.Add(name, node, gateway, orgID, cert, key, tlsCert, channel, contract, ca, caName, nil, autoGuest, autoRegister, autoRole, production)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Réseau %q configuré (id: %s).\n", name, n.ID)
	fmt.Fprintf(w, "Activez-le avec : myr network activate %s\n", n.ID)
	return nil
}

var networkAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Register a profile for an already-running network",
	Long: `Register a network profile pointing at a node that is already running
(started elsewhere, e.g. by another organisation, or manually) and persist
it in data/networks.json. Does not start or configure anything — for that,
use "myr network create" instead. The network is not activated automatically
after being added.

Examples:
  # Minimal (required flags only)
  myr network add --name sandbox --node node.org1.example.com:7051 --org-id Org1MSP

  # With channel, contract and certificate authority
  myr network add --name sandbox --node node.org1.example.com:7051 --org-id Org1MSP \
    --channel sandbox --contract myrcc \
    --ca https://ca.org1.example.com:7054 --ca-name ca-org1

  # Full configuration with client certificates
  myr network add --name prod --node node.org1.example.com:7051 --org-id Org1MSP \
    --channel main --contract myrcc \
    --cert data/certs/org1/client.pem --key data/certs/org1/client-key.pem \
    --tls-cert data/certs/org1/tls-root.pem`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetworkAdd(
			cmd.OutOrStdout(), networkSvc,
			networkAddName, networkAddNode, networkAddGateway, networkAddOrgID,
			networkAddCert, networkAddKey, networkAddTLSCert,
			networkAddChannel, networkAddContract, networkAddCA, networkAddCAName,
			networkAddAutoGuest, networkAddAutoRegister, networkAddAutoRole, networkAddProduction,
		)
	},
}

// ── update ────────────────────────────────────────────────────────────────────

func runNetworkUpdate(w io.Writer, cmd *cobra.Command, svc network.NetworkService, id string) error {
	networks, err := svc.List()
	if err != nil {
		return err
	}
	var existing *network.NetworkProfile
	for _, n := range networks {
		if n.ID == id {
			existing = n
			break
		}
	}
	if existing == nil {
		return fmt.Errorf("réseau introuvable : %s", id)
	}

	// Merge : seuls les flags explicitement fournis écrasent les valeurs existantes (DC-CLI-05).
	name := existing.Name
	peer := existing.PeerEndpoint
	gateway := existing.GatewayPeer
	msp := existing.MSPID
	cert := existing.CertPath
	key := existing.KeyPath
	tlsCert := existing.TLSCertPath
	channel := existing.FabricChannel
	chaincode := existing.GetChaincodeName()
	ca := existing.CAEndpoint
	caName := existing.CAName
	autoGuest := existing.AllowAutoGuest
	autoRegister := existing.AllowAutoRegister
	autoRole := existing.AutoRegisterRole
	production := existing.IsProduction

	if cmd.Flags().Changed("name") {
		if v, _ := cmd.Flags().GetString("name"); v != "" {
			name = v
		}
	}
	if cmd.Flags().Changed("node") {
		if v, _ := cmd.Flags().GetString("node"); v != "" {
			peer = v
		}
	}
	if cmd.Flags().Changed("gateway") {
		if v, _ := cmd.Flags().GetString("gateway"); v != "" {
			gateway = v
		}
	}
	if cmd.Flags().Changed("org-id") {
		if v, _ := cmd.Flags().GetString("org-id"); v != "" {
			msp = v
		}
	}
	if cmd.Flags().Changed("cert") {
		if v, _ := cmd.Flags().GetString("cert"); v != "" {
			cert = v
		}
	}
	if cmd.Flags().Changed("key") {
		if v, _ := cmd.Flags().GetString("key"); v != "" {
			key = v
		}
	}
	if cmd.Flags().Changed("tls-cert") {
		if v, _ := cmd.Flags().GetString("tls-cert"); v != "" {
			tlsCert = v
		}
	}
	if cmd.Flags().Changed("channel") {
		if v, _ := cmd.Flags().GetString("channel"); v != "" {
			channel = v
		}
	}
	if cmd.Flags().Changed("contract") {
		if v, _ := cmd.Flags().GetString("contract"); v != "" {
			chaincode = v
		}
	}
	if cmd.Flags().Changed("ca") {
		if v, _ := cmd.Flags().GetString("ca"); v != "" {
			ca = v
		}
	}
	if cmd.Flags().Changed("ca-name") {
		if v, _ := cmd.Flags().GetString("ca-name"); v != "" {
			caName = v
		}
	}
	if cmd.Flags().Changed("auto-guest") {
		autoGuest, _ = cmd.Flags().GetBool("auto-guest")
	}
	if cmd.Flags().Changed("auto-register") {
		autoRegister, _ = cmd.Flags().GetBool("auto-register")
	}
	if cmd.Flags().Changed("auto-role") {
		if v, _ := cmd.Flags().GetString("auto-role"); v != "" {
			autoRole = v
		}
	}
	if cmd.Flags().Changed("production") {
		production, _ = cmd.Flags().GetBool("production")
	}

	if _, err := svc.Update(id, name, peer, gateway, msp, cert, key, tlsCert, channel, chaincode, ca, caName, autoGuest, autoRegister, autoRole, production); err != nil {
		return err
	}
	fmt.Fprintf(w, "Réseau %s mis à jour.\n", id)
	return nil
}

var networkUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an existing network profile",
	Long: `Merge the provided flags into the existing profile.
Only explicitly passed flags override the existing values.

Examples:
  # Add client certificates
  myr network update <id> --cert ./certs/client.pem --key ./certs/client-key.pem

  # Change the contract name
  myr network update <id> --contract myrcc-v2

  # Mark as production (protects against destroy)
  myr network update <id> --production=true`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetworkUpdate(cmd.OutOrStdout(), cmd, networkSvc, args[0])
	},
}

// ── activate ──────────────────────────────────────────────────────────────────

func runNetworkActivate(w io.Writer, svc network.NetworkService, id string) error {
	networks, err := svc.List()
	if err != nil {
		return err
	}
	var name string
	for _, n := range networks {
		if n.ID == id {
			name = n.Name
			break
		}
	}
	if name == "" {
		return fmt.Errorf("réseau introuvable : %s", id)
	}
	if err := svc.Activate(id); err != nil {
		return err
	}
	fmt.Fprintf(w, "Réseau %q activé (id: %s).\n", name, id)
	return nil
}

var networkActivateCmd = &cobra.Command{
	Use:   "activate <id>",
	Short: "Set the active network",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetworkActivate(cmd.OutOrStdout(), networkSvc, args[0])
	},
}

// ── delete ────────────────────────────────────────────────────────────────────

func runNetworkDelete(w io.Writer, r io.Reader, svc network.NetworkService, id string, yes bool) error {
	networks, err := svc.List()
	if err != nil {
		return err
	}
	var target *network.NetworkProfile
	for _, n := range networks {
		if n.ID == id {
			target = n
			break
		}
	}
	if target == nil {
		return fmt.Errorf("réseau introuvable : %s", id)
	}
	if target.Active {
		fmt.Fprintln(w, "Avertissement : ce réseau est actuellement actif. La suppression désactive la connexion au réseau.")
	}
	if !yes {
		fmt.Fprintf(w, "Supprimer le réseau %q (%s) ? [o/N] : ", target.Name, id)
		scanner := bufio.NewScanner(r)
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "o" && answer != "oui" {
			fmt.Fprintln(w, "Annulé.")
			return nil
		}
	}
	if err := svc.Delete(id); err != nil {
		return err
	}
	if yes {
		fmt.Fprintf(w, "Réseau %q supprimé.\n", target.Name)
	} else {
		fmt.Fprintln(w, "Réseau supprimé.")
	}
	return nil
}

var networkDeleteYes bool

var networkDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a network profile",
	Long: `Remove the configuration profile. Does not stop blockchain processes
or delete local ledger data.

Examples:
  myr network delete net-1700000000000
  myr network delete net-1700000000000 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetworkDelete(cmd.OutOrStdout(), cmd.InOrStdin(), networkSvc, args[0], networkDeleteYes)
	},
}

// ── test ──────────────────────────────────────────────────────────────────────

func runNetworkTest(w io.Writer, svc network.NetworkService, id string) error {
	networks, err := svc.List()
	if err != nil {
		return err
	}
	targetID := id
	var targetEndpoint string
	if targetID == "" {
		for _, n := range networks {
			if n.Active {
				targetID = n.ID
				targetEndpoint = n.PeerEndpoint
				break
			}
		}
		if targetID == "" {
			return fmt.Errorf(`aucun réseau actif — activez-en un avec "myr network activate <id>"`)
		}
	} else {
		for _, n := range networks {
			if n.ID == id {
				targetEndpoint = n.PeerEndpoint
				break
			}
		}
		if targetEndpoint == "" {
			return fmt.Errorf("réseau introuvable : %s", id)
		}
	}
	fmt.Fprintf(w, "Test de connectivité vers %s...\n", targetEndpoint)
	if err := svc.TestConnection(targetID); err != nil {
		return fmt.Errorf("impossible de joindre %s — vérifiez le pare-feu et la disponibilité réseau", targetEndpoint)
	}
	fmt.Fprintln(w, "OK — nœud joignable")
	return nil
}

var networkTestCmd = &cobra.Command{
	Use:   "test [id]",
	Short: "Test TCP connectivity to the node",
	Long: `Test the network connection to the node endpoint of the specified profile.
If the id is omitted, the active network is used.

Examples:
  myr network test
  myr network test net-1700000000000`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := ""
		if len(args) > 0 {
			id = args[0]
		}
		return runNetworkTest(cmd.OutOrStdout(), networkSvc, id)
	},
}

// ── import ────────────────────────────────────────────────────────────────────

// fabricConnectionProfile représente un profil gateway-connection.json Fabric.
type fabricConnectionProfile struct {
	Name   string `json:"name"`
	Client struct {
		Organization string `json:"organization"`
	} `json:"client"`
	Organizations map[string]struct {
		MSPID string   `json:"mspid"`
		Peers []string `json:"peers"`
	} `json:"organizations"`
	Peers map[string]struct {
		URL         string `json:"url"`
		GRPCOptions struct {
			SSLTargetNameOverride string `json:"ssl-target-name-override"`
		} `json:"grpcOptions"`
		TLSCACerts struct {
			PEM string `json:"pem"`
		} `json:"tlsCACerts"`
	} `json:"peers"`
	CertificateAuthorities map[string]struct {
		URL    string `json:"url"`
		CAName string `json:"caName"`
	} `json:"certificateAuthorities"`
	Channels map[string]json.RawMessage `json:"channels"`
}

func runNetworkImport(w io.Writer, svc network.NetworkService, profilePath, nameOverride string, activate bool) error {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("fichier introuvable : %s", profilePath)
	}

	// 1. Tenter le format Fabric gateway-connection.json (DC-CLI-06)
	var fp fabricConnectionProfile
	if json.Unmarshal(data, &fp) == nil && fp.Client.Organization != "" {
		return importFabricProfile(w, svc, profilePath, fp, nameOverride, activate)
	}

	// 2. Tenter le format NetworkProfile JSON direct
	var np network.NetworkProfile
	if json.Unmarshal(data, &np) == nil && np.Name != "" {
		return importNetworkProfile(w, svc, np, nameOverride, activate)
	}

	return fmt.Errorf("format de profil invalide — ni Fabric connection profile ni NetworkProfile JSON reconnus")
}

func importFabricProfile(w io.Writer, svc network.NetworkService, profilePath string, p fabricConnectionProfile, nameOverride string, activate bool) error {
	org := p.Client.Organization
	orgData, ok := p.Organizations[org]
	if !ok {
		return fmt.Errorf("champ obligatoire manquant dans le profil : organizations.%s", org)
	}

	name := p.Name
	if nameOverride != "" {
		name = nameOverride
	}

	var peerEndpoint, gatewayPeer, tlsCertContent string
	if len(orgData.Peers) > 0 {
		peerName := orgData.Peers[0]
		if pd, ok := p.Peers[peerName]; ok {
			peerEndpoint = strings.TrimPrefix(pd.URL, "grpcs://")
			peerEndpoint = strings.TrimPrefix(peerEndpoint, "grpc://")
			gatewayPeer = pd.GRPCOptions.SSLTargetNameOverride
			tlsCertContent = pd.TLSCACerts.PEM
		}
	}

	// Premier canal de la section channels
	var fabricChannel string
	for ch := range p.Channels {
		fabricChannel = ch
		break
	}

	// Première CA
	var caEndpoint, caName string
	for _, ca := range p.CertificateAuthorities {
		caEndpoint = ca.URL
		caName = ca.CAName
		break
	}

	// Extraire le certificat TLS dans data/certs/<nom>/tls-root.pem
	var tlsCertPath string
	if tlsCertContent != "" {
		safeDir := filepath.Join("data", "certs", strings.ReplaceAll(name, " ", "-"))
		if err := os.MkdirAll(safeDir, 0o700); err == nil {
			tlsCertPath = filepath.Join(safeDir, "tls-root.pem")
			_ = os.WriteFile(tlsCertPath, []byte(tlsCertContent), 0o600)
		}
	}

	fmt.Fprintf(w, "Profil importé depuis %s\n", filepath.Base(profilePath))
	fmt.Fprintf(w, "  Nom        : %s\n", name)
	fmt.Fprintf(w, "  Org ID     : %s\n", orgData.MSPID)
	fmt.Fprintf(w, "  Nœud       : %s\n", peerEndpoint)
	fmt.Fprintf(w, "  Canal      : %s\n", fabricChannel)
	fmt.Fprintf(w, "  CA         : %s\n", caEndpoint)
	if tlsCertPath != "" {
		fmt.Fprintf(w, "  TLS cert   : %s (extrait du profil)\n", tlsCertPath)
	}
	fmt.Fprintln(w)

	n, err := svc.Add(name, peerEndpoint, gatewayPeer, orgData.MSPID, "", "", tlsCertPath, fabricChannel, "", caEndpoint, caName, nil, false, false, "", false)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ajout du profil : %w", err)
	}
	fmt.Fprintf(w, "Réseau importé (id: %s).\n", n.ID)
	if n.CertPath == "" || n.KeyPath == "" {
		fmt.Fprintf(w, "Complétez la configuration avec : myr network update %s --cert <cert> --key <key>\n", n.ID)
	}
	if activate {
		if err := svc.Activate(n.ID); err != nil {
			return fmt.Errorf("erreur lors de l'activation : %w", err)
		}
		fmt.Fprintf(w, "Réseau %q activé.\n", name)
	}
	return nil
}

func importNetworkProfile(w io.Writer, svc network.NetworkService, np network.NetworkProfile, nameOverride string, activate bool) error {
	name := np.Name
	if nameOverride != "" {
		name = nameOverride
	}
	n, err := svc.Add(name, np.PeerEndpoint, np.GatewayPeer, np.MSPID, np.CertPath, np.KeyPath, np.TLSCertPath, np.FabricChannel, np.GetChaincodeName(), np.CAEndpoint, np.CAName, np.Channels,
		np.AllowAutoGuest, np.AllowAutoRegister, np.AutoRegisterRole, np.IsProduction)
	if err != nil {
		return fmt.Errorf("erreur lors de l'importation : %w", err)
	}
	fmt.Fprintf(w, "Réseau importé (id: %s).\n", n.ID)
	if activate {
		if err := svc.Activate(n.ID); err != nil {
			return fmt.Errorf("erreur lors de l'activation : %w", err)
		}
		fmt.Fprintf(w, "Réseau %q activé.\n", name)
	}
	return nil
}

var (
	networkImportProfile  string
	networkImportName     string
	networkImportActivate bool
)

var networkImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a network profile from a JSON file",
	Long: `Import a network profile from a JSON file.
Supports two formats:
  1. Blockchain gateway connection profile (extracts org ID, node, authority, TLS cert)
  2. Myr network profile JSON (direct export)

Examples:
  myr network import --profile connection-org1.json
  myr network import --profile connection-org1.json --activate
  myr network import --profile network-backup.json --name my-network`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNetworkImport(cmd.OutOrStdout(), networkSvc, networkImportProfile, networkImportName, networkImportActivate)
	},
}

// ── destroy ───────────────────────────────────────────────────────────────────

func runNetworkDestroy(w io.Writer, svc network.NetworkService, id string, confirm bool, dataPath string) error {
	networks, err := svc.List()
	if err != nil {
		return err
	}
	var target *network.NetworkProfile
	for _, n := range networks {
		if n.ID == id {
			target = n
			break
		}
	}
	if target == nil {
		return fmt.Errorf("réseau introuvable : %s", id)
	}

	if !confirm {
		fmt.Fprintf(w, "ATTENTION : Cette opération est irréversible.\n")
		fmt.Fprintf(w, "Elle supprimera toutes les données locales du réseau %q.\n", target.Name)
		fmt.Fprintf(w, "  - Processus blockchain (nœuds, CA) arrêtés\n")
		fmt.Fprintf(w, "  - Données ledger supprimées (%s)\n", dataPath)
		fmt.Fprintf(w, "  - Artefacts cryptographiques supprimés (data/certs/%s/)\n", id)
		fmt.Fprintf(w, "  - Profil réseau supprimé\n\n")
		fmt.Fprintf(w, "Utilisez --confirm pour confirmer :\n")
		fmt.Fprintf(w, "  myr network destroy %s --confirm\n", id)
		return nil
	}

	if target.IsProduction {
		return fmt.Errorf(
			"%q est marqué comme réseau de production (IsProduction: true).\nDémantèlement refusé. Pour forcer, retirez le flag production :\n  myr network update %s --production=false",
			target.Name, id,
		)
	}

	// Arrêt des processus Fabric (DC-CLI-07 : Docker > systemd > pkill)
	stopFabricProcesses(w)

	// Suppression des données ledger
	if dataPath != "" {
		if err := os.RemoveAll(dataPath); err != nil {
			fmt.Fprintf(w, "Avertissement : impossible de supprimer %s : %v\n", dataPath, err)
		} else {
			fmt.Fprintln(w, "Données ledger supprimées.")
		}
	}

	// Suppression des artefacts cryptographiques
	cryptoPath := filepath.Join("data", "certs", id)
	if err := os.RemoveAll(cryptoPath); err != nil {
		fmt.Fprintf(w, "Avertissement : impossible de supprimer %s : %v\n", cryptoPath, err)
	} else {
		fmt.Fprintln(w, "Artefacts cryptographiques supprimés.")
	}

	if err := svc.Delete(id); err != nil {
		return err
	}
	fmt.Fprintf(w, "Réseau %q démantelé. Toutes les données locales ont été supprimées.\n", target.Name)
	return nil
}

// stopFabricProcesses arrête les processus Fabric dans l'ordre : Docker Compose > systemd > pkill (DC-CLI-07).
func stopFabricProcesses(w io.Writer) {
	composePath := os.Getenv("MYR_FABRIC_COMPOSE_PATH")
	if composePath == "" {
		composePath = "data/docker-compose.yaml"
	}
	if _, err := os.Stat(composePath); err == nil {
		if err := exec.Command("docker", "compose", "-f", composePath, "down").Run(); err == nil {
			fmt.Fprintln(w, "Conteneurs blockchain arrêtés (Docker Compose).")
			return
		}
	}
	if _, err := os.Stat("/etc/systemd/system"); err == nil {
		if err := exec.Command("systemctl", "stop", "fabric-peer", "fabric-orderer", "fabric-ca").Run(); err == nil {
			fmt.Fprintln(w, "Services blockchain arrêtés (systemd).")
			return
		}
	}
	for _, proc := range []string{"peer", "orderer", "fabric-ca-server"} {
		if err := exec.Command("pkill", "-SIGTERM", proc).Run(); err != nil {
			fmt.Fprintf(w, "Avertissement : processus %s non arrêté — poursuite de la suppression.\n", proc)
		}
	}
}

var (
	networkDestroyConfirm  bool
	networkDestroyDataPath string
)

var networkDestroyCmd = &cobra.Command{
	Use:   "destroy <id>",
	Short: "Dismantle a dev/test network (irreversible)",
	Long: `Stop blockchain processes, delete local ledger data and cryptographic
artefacts, then remove the network profile. This operation is irreversible.

Protected against production networks (--production=true).
Reserved for dev/test networks only.

Examples:
  myr network destroy <id> --confirm
  myr network destroy <id> --confirm --data-path /var/hyperledger/production`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dataPath := networkDestroyDataPath
		if dataPath == "" {
			dataPath = os.Getenv("MYR_FABRIC_DATA_PATH")
		}
		if dataPath == "" {
			dataPath = "/var/hyperledger/production"
		}
		return runNetworkDestroy(cmd.OutOrStdout(), networkSvc, args[0], networkDestroyConfirm, dataPath)
	},
}

// ── resolveChannel ────────────────────────────────────────────────────────────

// resolveChannel résout le canal cible pour les commandes admin (DC-CLI-01).
// Utilise channelFlag si fourni, sinon le FabricChannel du réseau actif.
func resolveChannel(channelFlag string, svc network.NetworkService) (string, error) {
	if channelFlag != "" {
		return channelFlag, nil
	}
	active, err := svc.GetActive()
	if err != nil {
		return "", err
	}
	if active == nil || active.FabricChannel == "" {
		return "", fmt.Errorf(`canal non configuré — utilisez --channel ou myr network update --channel <id>`)
	}
	return active.FabricChannel, nil
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	networkCmd.AddCommand(
		networkListCmd,
		networkShowCmd,
		networkAddCmd,
		networkUpdateCmd,
		networkActivateCmd,
		networkDeleteCmd,
		networkTestCmd,
		networkImportCmd,
		networkDestroyCmd,
	)

	// add flags
	networkAddCmd.Flags().StringVar(&networkAddName, "name", "", "Nom lisible du réseau (requis)")
	networkAddCmd.Flags().StringVar(&networkAddNode, "node", "", "Adresse du nœud d'entrée host:port (requis)")
	networkAddCmd.Flags().StringVar(&networkAddOrgID, "org-id", "", "Identifiant de l'organisation (requis)")
	networkAddCmd.Flags().StringVar(&networkAddGateway, "gateway", "", "Nom TLS du nœud d'entrée")
	networkAddCmd.Flags().StringVar(&networkAddCert, "cert", "", "Chemin vers le certificat client PEM")
	networkAddCmd.Flags().StringVar(&networkAddKey, "key", "", "Chemin vers la clé privée PEM")
	networkAddCmd.Flags().StringVar(&networkAddTLSCert, "tls-cert", "", "Chemin vers le certificat TLS du nœud PEM")
	networkAddCmd.Flags().StringVar(&networkAddChannel, "channel", "", "Canal blockchain par défaut")
	networkAddCmd.Flags().StringVar(&networkAddContract, "contract", "", "Nom du contrat déployé")
	networkAddCmd.Flags().StringVar(&networkAddCA, "ca", "", "URL de l'autorité de certification (https://host:port)")
	networkAddCmd.Flags().StringVar(&networkAddCAName, "ca-name", "", "Nom de l'autorité de certification")
	networkAddCmd.Flags().BoolVar(&networkAddAutoGuest, "auto-guest", false, "Autoriser les accès invité automatiques")
	networkAddCmd.Flags().BoolVar(&networkAddAutoRegister, "auto-register", false, "Autoriser l'enregistrement CA automatique")
	networkAddCmd.Flags().StringVar(&networkAddAutoRole, "auto-role", "reader", "Rôle auto-register : reader, contributor ou auditor")
	networkAddCmd.Flags().BoolVar(&networkAddProduction, "production", false, "Marquer ce réseau comme réseau de production — protège contre destroy")
	_ = networkAddCmd.MarkFlagRequired("name")
	_ = networkAddCmd.MarkFlagRequired("node")
	_ = networkAddCmd.MarkFlagRequired("org-id")

	// update flags (tous optionnels)
	networkUpdateCmd.Flags().String("name", "", "Nom lisible du réseau")
	networkUpdateCmd.Flags().String("node", "", "Adresse du nœud d'entrée host:port")
	networkUpdateCmd.Flags().String("org-id", "", "Identifiant de l'organisation")
	networkUpdateCmd.Flags().String("gateway", "", "Nom TLS du nœud d'entrée")
	networkUpdateCmd.Flags().String("cert", "", "Chemin vers le certificat client PEM")
	networkUpdateCmd.Flags().String("key", "", "Chemin vers la clé privée PEM")
	networkUpdateCmd.Flags().String("tls-cert", "", "Chemin vers le certificat TLS du nœud PEM")
	networkUpdateCmd.Flags().String("channel", "", "Canal blockchain par défaut")
	networkUpdateCmd.Flags().String("contract", "", "Nom du contrat déployé")
	networkUpdateCmd.Flags().String("ca", "", "URL de l'autorité de certification")
	networkUpdateCmd.Flags().String("ca-name", "", "Nom de l'autorité de certification")
	networkUpdateCmd.Flags().Bool("auto-guest", false, "Autoriser les accès invité automatiques")
	networkUpdateCmd.Flags().Bool("auto-register", false, "Autoriser l'enregistrement CA automatique")
	networkUpdateCmd.Flags().String("auto-role", "", "Rôle auto-register : reader, contributor ou auditor")
	networkUpdateCmd.Flags().Bool("production", false, "Marquer ce réseau comme réseau de production — protège contre destroy")

	// delete flags
	networkDeleteCmd.Flags().BoolVar(&networkDeleteYes, "yes", false, "Ignorer la demande de confirmation interactive")

	// import flags
	networkImportCmd.Flags().StringVar(&networkImportProfile, "profile", "", "Chemin vers le fichier JSON du profil (requis)")
	networkImportCmd.Flags().StringVar(&networkImportName, "name", "", "Surcharge le nom importé depuis le fichier")
	networkImportCmd.Flags().BoolVar(&networkImportActivate, "activate", false, "Activer automatiquement le réseau importé")
	_ = networkImportCmd.MarkFlagRequired("profile")

	// destroy flags
	networkDestroyCmd.Flags().BoolVar(&networkDestroyConfirm, "confirm", false, "Confirmation explicite obligatoire (DC-CLI-04)")
	networkDestroyCmd.Flags().StringVar(&networkDestroyDataPath, "data-path", "", "Chemin du répertoire ledger Fabric local à supprimer")
}
