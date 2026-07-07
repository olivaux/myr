// adapters/in/cli/identity.go — gestion des identités (rôle utilisateur)
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage user identities registered with the certificate authority",
}

// ── set-role ──────────────────────────────────────────────────────────────────

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
		if err := identitySvc.SetRole(context.Background(), identitySetRoleID, identitySetRoleName); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Rôle de %q mis à jour : %s.\n", identitySetRoleID, identitySetRoleName)
		fmt.Fprintln(cmd.OutOrStdout(), "Le nouveau rôle s'applique au prochain ré-enrôlement de l'identité.")
		return nil
	},
}

func init() {
	identityCmd.AddCommand(identitySetRoleCmd)

	identitySetRoleCmd.Flags().StringVar(&identitySetRoleID, "id", "", "Identité CA cible, ex: alice@org1 (requis)")
	identitySetRoleCmd.Flags().StringVar(&identitySetRoleName, "role", "", "Nouveau rôle (requis)")
	_ = identitySetRoleCmd.MarkFlagRequired("id")
	_ = identitySetRoleCmd.MarkFlagRequired("role")
}
