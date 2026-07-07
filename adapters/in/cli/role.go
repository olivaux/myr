// adapters/in/cli/role.go — gestion des rôles RBAC (permissions personnalisées)
package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"myr/domain/role"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage RBAC roles and their permissions",
	Long: `Commands to create, inspect and manage roles and the permissions attached
to them. The 4 built-in roles (reader, contributor, auditor, admin) always
exist and cannot be modified or deleted. Custom roles (e.g. consumer,
manufacturer) can be created with any subset of the permission catalogue.

Permissions: read, write, network.admin, role.admin, identity.admin, admin.`,
}

// ── list ──────────────────────────────────────────────────────────────────────

func runRoleList(w io.Writer, svc role.RoleService) error {
	roles, err := svc.List()
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NOM\tINTÉGRÉ\tPERMISSIONS")
	for _, r := range roles {
		perms := make([]string, len(r.Permissions))
		for i, p := range r.Permissions {
			perms[i] = string(p)
		}
		builtIn := ""
		if r.BuiltIn {
			builtIn = "*"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", r.Name, builtIn, strings.Join(perms, ", "))
	}
	return tw.Flush()
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all roles and their permissions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRoleList(cmd.OutOrStdout(), roleSvc)
	},
}

// ── show ──────────────────────────────────────────────────────────────────────

var roleShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a single role's permissions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := roleSvc.Get(args[0])
		if err != nil {
			return err
		}
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Rôle : %s\n", r.Name)
		fmt.Fprintf(w, "Intégré : %v\n", r.BuiltIn)
		fmt.Fprintln(w, "Permissions :")
		for _, p := range r.Permissions {
			fmt.Fprintf(w, "  - %s\n", p)
		}
		return nil
	},
}

// ── create ────────────────────────────────────────────────────────────────────

var roleCreatePerms []string

var roleCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a custom role with a given set of permissions",
	Long: `Create a new role with an arbitrary subset of the permission catalogue.

Examples:
  myr role create consumer --permission read
  myr role create manufacturer --permission read --permission write`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		perms := make([]role.Permission, len(roleCreatePerms))
		for i, p := range roleCreatePerms {
			perms[i] = role.Permission(p)
		}
		r, err := roleSvc.Create(args[0], perms)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Rôle %q créé.\n", r.Name)
		return nil
	},
}

// ── update ────────────────────────────────────────────────────────────────────

var roleUpdatePerms []string

var roleUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Replace the permission set of an existing custom role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		perms := make([]role.Permission, len(roleUpdatePerms))
		for i, p := range roleUpdatePerms {
			perms[i] = role.Permission(p)
		}
		r, err := roleSvc.Update(args[0], perms)
		if err != nil {
			if err == role.ErrBuiltIn {
				return fmt.Errorf("%q est un rôle intégré — non modifiable", args[0])
			}
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Rôle %q mis à jour.\n", r.Name)
		return nil
	},
}

// ── delete ────────────────────────────────────────────────────────────────────

var roleDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a custom role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := roleSvc.Delete(args[0]); err != nil {
			if err == role.ErrBuiltIn {
				return fmt.Errorf("%q est un rôle intégré — non supprimable", args[0])
			}
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Rôle %q supprimé.\n", args[0])
		return nil
	},
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	roleCmd.AddCommand(roleListCmd, roleShowCmd, roleCreateCmd, roleUpdateCmd, roleDeleteCmd)

	roleCreateCmd.Flags().StringArrayVar(&roleCreatePerms, "permission", nil, "Permission à accorder (répétable) : read, write, network.admin, role.admin, identity.admin, admin")
	roleUpdateCmd.Flags().StringArrayVar(&roleUpdatePerms, "permission", nil, "Nouvel ensemble de permissions (répétable, remplace l'existant)")
}
