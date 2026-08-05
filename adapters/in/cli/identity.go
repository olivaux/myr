// adapters/in/cli/identity.go — gestion des identités (compte CA, rôle utilisateur)
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"myr-core/domain/identity"
	"myr-core/domain/network"
)

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage user identities registered with the certificate authority",
}

// ── wallets ───────────────────────────────────────────────────────────────────

func runIdentityWallets(w io.Writer, svc identity.IdentityService) error {
	wallets, err := svc.ListLocalWallets()
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "HANDLE\tPSEUDO\tORG\tSTATUT")
	for _, wa := range wallets {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", wa.Handle, wa.Name, wa.OrgID, wa.Status)
	}
	return tw.Flush()
}

var identityWalletsCmd = &cobra.Command{
	Use:   "wallets",
	Short: "List wallets present locally (~/.Myr/wallets/)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if identitySvc == nil {
			return fmt.Errorf("service identité non disponible")
		}
		return runIdentityWallets(cmd.OutOrStdout(), identitySvc)
	},
}

// ── status ────────────────────────────────────────────────────────────────────

func splitIdentityHandle(handle string) (name, orgID string, err error) {
	parts := strings.SplitN(handle, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("handle invalide — format attendu : pseudo@org")
	}
	return parts[0], parts[1] + "MSP", nil
}

func runIdentityStatus(w io.Writer, svc identity.IdentityService, handle string) error {
	name, orgID, err := splitIdentityHandle(handle)
	if err != nil {
		return err
	}
	status, err := svc.GetStatus(context.Background(), identity.WalletEntry{
		Handle: handle, Name: name, OrgID: orgID,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s : %s\n", handle, status)
	return nil
}

var identityStatusHandle string

var identityStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Query the CA status of a wallet",
	Long: `Query the certificate authority for the current status of an identity.

Example:
  myr identity status --handle alice@Org1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if identitySvc == nil {
			return fmt.Errorf("service identité non disponible")
		}
		return runIdentityStatus(cmd.OutOrStdout(), identitySvc, identityStatusHandle)
	},
}

// ── enroll ────────────────────────────────────────────────────────────────────

func runIdentityEnroll(w io.Writer, svc identity.IdentityService, name, secret, orgID string) error {
	wa, err := svc.Enroll(context.Background(), name, secret, orgID)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Identité enrôlée : %s  statut=%s\n", wa.Handle, wa.Status)
	return nil
}

var (
	identityEnrollName   string
	identityEnrollSecret string
	identityEnrollOrgID  string
)

var identityEnrollCmd = &cobra.Command{
	Use:   "enroll",
	Short: "Enroll with the CA using an enrollment secret and save the wallet locally",
	Long: `Enroll an identity already known to the certificate authority, using the
enrollment secret it was given, and save the resulting wallet under
~/.Myr/wallets/.

Example:
  myr identity enroll --name alice --secret <secret> --org-id Org1MSP`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if identitySvc == nil {
			return fmt.Errorf("service identité non disponible")
		}
		return runIdentityEnroll(cmd.OutOrStdout(), identitySvc, identityEnrollName, identityEnrollSecret, identityEnrollOrgID)
	},
}

// ── re-enroll ─────────────────────────────────────────────────────────────────

func runIdentityReEnroll(w io.Writer, svc identity.IdentityService, handle string) error {
	wallets, err := svc.ListLocalWallets()
	if err != nil {
		return err
	}
	var found *identity.WalletEntry
	for i := range wallets {
		if wallets[i].Handle == handle {
			found = &wallets[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("aucun wallet local pour %q — utilisez d'abord 'myr identity enroll'", handle)
	}
	wa, err := svc.ReEnroll(context.Background(), *found)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Identité %s ré-enrôlée : statut=%s\n", wa.Handle, wa.Status)
	return nil
}

var identityReEnrollCmd = &cobra.Command{
	Use:   "re-enroll <handle>",
	Short: "Renew the certificate of a locally known wallet",
	Long: `Re-enroll a wallet already present locally, issuing a fresh certificate.
Required for a role change (myr identity set-role) to take effect in the
active certificate.

Example:
  myr identity re-enroll alice@Org1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if identitySvc == nil {
			return fmt.Errorf("service identité non disponible")
		}
		return runIdentityReEnroll(cmd.OutOrStdout(), identitySvc, args[0])
	},
}

// ── request ───────────────────────────────────────────────────────────────────

func runIdentityRequest(w io.Writer, isvc identity.IdentityService, nsvc network.NetworkService, req identity.AccountRequest) error {
	saved, err := isvc.SubmitRequest(req)
	if err != nil {
		return err
	}

	if nsvc != nil {
		if profile, err2 := nsvc.GetActive(); err2 == nil && profile != nil && profile.AllowAutoRegister {
			secret, err3 := isvc.AutoRegister(context.Background(), saved, profile.AutoRegisterRole)
			if err3 == nil {
				fmt.Fprintf(w, "Identité créée automatiquement : %s\nSecret d'enrôlement : %s\n", saved.Pseudo, secret)
				return nil
			}
		}
	}
	fmt.Fprintf(w, "Demande enregistrée (id=%s, statut=%s) — un administrateur doit créer l'identité manuellement.\n", saved.ID, saved.Status)
	return nil
}

var (
	identityRequestPseudo      string
	identityRequestDisplayName string
	identityRequestEmail       string
	identityRequestOrgID       string
	identityRequestMessage     string
)

var identityRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Submit an access request on behalf of a user",
	Long: `Submit an access request for a pseudo/email/organisation, on behalf of a
user who does not have an account yet — the CLI equivalent of
POST /api/identity/request.

If the active network profile allows auto-registration, the identity is
created immediately and the enrollment secret is printed. Otherwise the
request is only recorded as pending — an administrator must create the
identity in the CA out of band and transmit the secret separately.

Example:
  myr identity request --pseudo alice --email alice@example.com --org-id Org1MSP`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if identitySvc == nil {
			return fmt.Errorf("service identité non disponible")
		}
		req := identity.AccountRequest{
			Pseudo:      identityRequestPseudo,
			DisplayName: identityRequestDisplayName,
			Email:       identityRequestEmail,
			OrgID:       identityRequestOrgID,
			Message:     identityRequestMessage,
		}
		return runIdentityRequest(cmd.OutOrStdout(), identitySvc, networkSvc, req)
	},
}

// ── requests ──────────────────────────────────────────────────────────────────

func runIdentityRequests(w io.Writer, svc identity.IdentityService) error {
	reqs, err := svc.ListRequests()
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tPSEUDO\tEMAIL\tORG\tSTATUT")
	for _, r := range reqs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.Pseudo, r.Email, r.OrgID, r.Status)
	}
	return tw.Flush()
}

var identityRequestsCmd = &cobra.Command{
	Use:   "requests",
	Short: "List pending access requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		if identitySvc == nil {
			return fmt.Errorf("service identité non disponible")
		}
		return runIdentityRequests(cmd.OutOrStdout(), identitySvc)
	},
}

// ── set-role ──────────────────────────────────────────────────────────────────

func runIdentitySetRole(w io.Writer, svc identity.IdentityService, id, role string) error {
	if err := svc.SetRole(context.Background(), id, role); err != nil {
		return err
	}
	fmt.Fprintf(w, "Rôle de %q mis à jour : %s.\n", id, role)
	fmt.Fprintln(w, "Le nouveau rôle s'applique au prochain ré-enrôlement de l'identité.")
	return nil
}

var (
	identitySetRoleID   string
	identitySetRoleName string
)

var identitySetRoleCmd = &cobra.Command{
	Use:   "set-role",
	Short: "Change the role of an existing identity",
	Long: `Change the role attribute (Myr.role) of an identity already registered
with the certificate authority.

The new role only applies to certificates issued after this call — the
identity must re-enroll for the new role to take effect in its active
certificate. This is a property of the certificate authority, not a myr
limitation.

Examples:
  myr identity set-role --id alice@org1 --role contributor
  myr identity set-role --id bob@org1 --role consumer`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if identitySvc == nil {
			return fmt.Errorf("service identité non disponible")
		}
		return runIdentitySetRole(cmd.OutOrStdout(), identitySvc, identitySetRoleID, identitySetRoleName)
	},
}

func init() {
	identityCmd.AddCommand(
		identityWalletsCmd, identityStatusCmd, identityEnrollCmd, identityReEnrollCmd,
		identityRequestCmd, identityRequestsCmd, identitySetRoleCmd,
	)

	identityStatusCmd.Flags().StringVar(&identityStatusHandle, "handle", "", "Identité CA, ex: alice@Org1 (requis)")
	_ = identityStatusCmd.MarkFlagRequired("handle")

	identityEnrollCmd.Flags().StringVar(&identityEnrollName, "name", "", "Pseudo (requis)")
	identityEnrollCmd.Flags().StringVar(&identityEnrollSecret, "secret", "", "Secret d'enrôlement fourni par l'admin (requis)")
	identityEnrollCmd.Flags().StringVar(&identityEnrollOrgID, "org-id", "", "ID de l'organisation, ex: Org1MSP (requis)")
	_ = identityEnrollCmd.MarkFlagRequired("name")
	_ = identityEnrollCmd.MarkFlagRequired("secret")
	_ = identityEnrollCmd.MarkFlagRequired("org-id")

	identityRequestCmd.Flags().StringVar(&identityRequestPseudo, "pseudo", "", "Pseudo souhaité (requis)")
	identityRequestCmd.Flags().StringVar(&identityRequestDisplayName, "display-name", "", "Nom affiché (optionnel)")
	identityRequestCmd.Flags().StringVar(&identityRequestEmail, "email", "", "E-mail (requis)")
	identityRequestCmd.Flags().StringVar(&identityRequestOrgID, "org-id", "", "Organisation souhaitée, ex: Org1MSP (requis)")
	identityRequestCmd.Flags().StringVar(&identityRequestMessage, "message", "", "Message libre pour l'admin (optionnel)")
	_ = identityRequestCmd.MarkFlagRequired("pseudo")
	_ = identityRequestCmd.MarkFlagRequired("email")
	_ = identityRequestCmd.MarkFlagRequired("org-id")

	identitySetRoleCmd.Flags().StringVar(&identitySetRoleID, "id", "", "Identité CA cible, ex: alice@org1 (requis)")
	identitySetRoleCmd.Flags().StringVar(&identitySetRoleName, "role", "", "Nouveau rôle (requis)")
	_ = identitySetRoleCmd.MarkFlagRequired("id")
	_ = identitySetRoleCmd.MarkFlagRequired("role")
}
