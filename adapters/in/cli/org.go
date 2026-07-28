// adapters/in/cli/org.go — commandes de gestion des organisations (UCADM01)
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"myr-core/domain/channel"
	"myr-core/domain/network"
)

// ── myr org ───────────────────────────────────────────────────────────────────

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage member organisations in a network channel",
	Long: `Commands to administer the member organisations of a blockchain channel.

Adding an organisation to the channel requires administrator privileges
and that the endorsement policy is satisfied by the existing admins.`,
}

// ── org add ───────────────────────────────────────────────────────────────────

var (
	orgAddMSP     string
	orgAddName    string
	orgAddCert    string
	orgAddChannel string
	orgAddRole    string
	orgAddTLSCert string
	orgAddUpdate  bool
)

func runOrgAdd(w io.Writer, chSvc channel.ChannelService, netSvc network.NetworkService) error {
	certData, err := os.ReadFile(orgAddCert)
	if err != nil {
		return fmt.Errorf("certificat introuvable : %s", orgAddCert)
	}

	var tlsCertData string
	if orgAddTLSCert != "" {
		data, err := os.ReadFile(orgAddTLSCert)
		if err != nil {
			return fmt.Errorf("certificat TLS introuvable : %s", orgAddTLSCert)
		}
		tlsCertData = string(data)
	}

	channelID, err := resolveChannel(orgAddChannel, netSvc)
	if err != nil {
		return err
	}

	role := orgAddRole
	if role == "" {
		role = "member"
	}

	org := channel.Organization{
		MSPID:    orgAddMSP,
		Name:     orgAddName,
		Role:     role,
		RootCert: string(certData),
		TLSCert:  tlsCertData,
	}

	err = chSvc.AddOrganisation(channelID, org)
	if err != nil {
		if errors.Is(err, channel.ErrAlreadyMember) {
			fmt.Fprintf(w, "L'organisation %s est déjà membre du canal %s.\n", orgAddMSP, channelID)
			if !orgAddUpdate {
				return fmt.Errorf("utilisez --update pour forcer la mise à jour")
			}
			fmt.Fprintln(w, "Mise à jour du rôle et de la politique d'accès...")
			if err2 := chSvc.AddOrganisation(channelID, org); err2 != nil {
				return err2
			}
			fmt.Fprintf(w, "Organisation %q mise à jour sur le canal %s.\n", orgAddName, channelID)
			return nil
		}
		if errors.Is(err, channel.ErrInvalidMSPID) {
			return fmt.Errorf("identifiant d'organisation invalide — caractères non autorisés ou longueur hors limites (max 128)")
		}
		if errors.Is(err, channel.ErrFabricUnavailable) {
			return fmt.Errorf("adaptateur blockchain non configuré — vérifiez le réseau actif et les variables d'environnement")
		}
		if errors.Is(err, channel.ErrEndorsementPolicy) {
			return fmt.Errorf("politique d'endorsement non satisfaite. Contactez les autres administrateurs d'organisation")
		}
		return err
	}

	fmt.Fprintf(w, "Organisation %q (id: %s) ajoutée au canal %s.\n", orgAddName, orgAddMSP, channelID)
	fmt.Fprintln(w, "La configuration du canal est mise à jour sur le réseau.")
	return nil
}

var orgAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an organisation to the network channel",
	Long: `Add an organisation to the active channel or to the channel
specified with --channel. Validates the organisation identifier format before submission.

Examples:
  myr org add --org-id Org2MSP --name "Org 2" --cert org2-ca.pem
  myr org add --org-id Org2MSP --name "Org 2" --cert org2-ca.pem --channel sandbox --role admin`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOrgAdd(cmd.OutOrStdout(), channelSvc, networkSvc)
	},
}

func init() {
	orgCmd.AddCommand(orgAddCmd)

	orgAddCmd.Flags().StringVar(&orgAddMSP, "org-id", "", "Identifiant de l'organisation (requis)")
	orgAddCmd.Flags().StringVar(&orgAddName, "name", "", "Nom lisible de l'organisation (requis)")
	orgAddCmd.Flags().StringVar(&orgAddCert, "cert", "", "Chemin vers le certificat CA racine PEM (requis)")
	orgAddCmd.Flags().StringVar(&orgAddChannel, "channel", "", "ID du canal cible (défaut: canal du réseau actif)")
	orgAddCmd.Flags().StringVar(&orgAddRole, "role", "member", "Rôle : member ou admin")
	orgAddCmd.Flags().StringVar(&orgAddTLSCert, "tls-cert", "", "Chemin vers le certificat TLS CA racine PEM")
	orgAddCmd.Flags().BoolVar(&orgAddUpdate, "update", false, "Forcer la mise à jour si l'organisation est déjà membre")
	_ = orgAddCmd.MarkFlagRequired("org-id")
	_ = orgAddCmd.MarkFlagRequired("name")
	_ = orgAddCmd.MarkFlagRequired("cert")
}
