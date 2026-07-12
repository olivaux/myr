// adapters/in/cli/session.go — compte local CLI (bootstrap machine, un seul enregistrement)
//
// À ne pas confondre avec les sessions REST (adapters/in/rest/session.go) : ce
// domaine décrit le compte local de la machine qui exécute le CLI, pas un
// jeton par connexion HTTP.
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"myr/domain/session"
)

var sessionSvc session.SessionService

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage the local CLI machine account (first-run bootstrap)",
	Long: `Commands to create, inspect and clear the local account bootstrapped on
this machine the first time the CLI is used. There is at most one such
account per machine — access rights themselves are managed on the network
side by the administrator (myr role, myr identity set-role).`,
}

// ── create ────────────────────────────────────────────────────────────────────

func runSessionCreate(w io.Writer, svc session.SessionService, name, orgID string) error {
	s, err := svc.Create(name, orgID)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Compte local créé : %s (org: %s)\n", s.Name, s.OrgID)
	return nil
}

var (
	sessionCreateName  string
	sessionCreateOrgID string
)

var sessionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Bootstrap the local account on this machine",
	Long: `Create the local machine account (name + organisation). Fails if an
account already exists — use "myr session logout" first to replace it.

Example:
  myr session create --name alice --org-id Org1MSP`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if sessionSvc == nil {
			return fmt.Errorf("service session non disponible")
		}
		return runSessionCreate(cmd.OutOrStdout(), sessionSvc, sessionCreateName, sessionCreateOrgID)
	},
}

// ── show ──────────────────────────────────────────────────────────────────────

func runSessionShow(w io.Writer, svc session.SessionService) error {
	s, err := svc.Current()
	if err != nil {
		return err
	}
	if s == nil {
		fmt.Fprintln(w, "Aucun compte local — utilisez 'myr session create'.")
		return nil
	}
	fmt.Fprintf(w, "Nom       : %s\n", s.Name)
	fmt.Fprintf(w, "Org       : %s\n", s.OrgID)
	fmt.Fprintf(w, "Créé le   : %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

var sessionShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the local machine account, if any",
	RunE: func(cmd *cobra.Command, args []string) error {
		if sessionSvc == nil {
			return fmt.Errorf("service session non disponible")
		}
		return runSessionShow(cmd.OutOrStdout(), sessionSvc)
	},
}

// ── logout ────────────────────────────────────────────────────────────────────

func runSessionLogout(w io.Writer, svc session.SessionService) error {
	if err := svc.Logout(); err != nil {
		return err
	}
	fmt.Fprintln(w, "Compte local effacé.")
	return nil
}

var sessionLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the local machine account",
	RunE: func(cmd *cobra.Command, args []string) error {
		if sessionSvc == nil {
			return fmt.Errorf("service session non disponible")
		}
		return runSessionLogout(cmd.OutOrStdout(), sessionSvc)
	},
}

func init() {
	sessionCreateCmd.Flags().StringVar(&sessionCreateName, "name", "", "Nom du compte local (requis)")
	sessionCreateCmd.Flags().StringVar(&sessionCreateOrgID, "org-id", "", "Organisation associée (optionnel)")
	_ = sessionCreateCmd.MarkFlagRequired("name")

	sessionCmd.AddCommand(sessionCreateCmd, sessionShowCmd, sessionLogoutCmd)
}
