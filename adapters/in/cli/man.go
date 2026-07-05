// adapters/in/cli/man.go — built-in man page renderer for myr commands
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

var manCmd = &cobra.Command{
	Use:   "man [command...]",
	Short: "Display the man page for a myr command",
	Long: `Display formatted manual page documentation for any myr command.
Without arguments, shows the man page for the root myr command.

Examples:
  myr man
  myr man network
  myr man network add
  myr man org add
  myr man node remove`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runManPage(cmd.OutOrStdout(), cmd.Root(), strings.Join(args, " "))
	},
}

func runManPage(w io.Writer, root *cobra.Command, cmdPath string) error {
	target := findCommand(root, strings.Fields(cmdPath))
	if target == nil {
		return fmt.Errorf("unknown command %q — run 'myr man' for a list of top-level commands", cmdPath)
	}
	renderManPage(w, target)
	return nil
}

// findCommand traverses the cobra command tree from root following path parts.
func findCommand(root *cobra.Command, parts []string) *cobra.Command {
	if len(parts) == 0 {
		return root
	}
	cmd := root
	for _, part := range parts {
		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == part {
				cmd = sub
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return cmd
}

// renderManPage writes man-page style documentation for cmd to w.
func renderManPage(w io.Writer, cmd *cobra.Command) {
	sep := strings.Repeat("─", 60)
	fmt.Fprintln(w, sep)

	// NAME
	fmt.Fprintln(w, "NAME")
	fmt.Fprintf(w, "       %s -- %s\n\n", cmd.CommandPath(), cmd.Short)

	// SYNOPSIS
	fmt.Fprintln(w, "SYNOPSIS")
	fmt.Fprintf(w, "       %s\n\n", cmd.UseLine())

	// DESCRIPTION
	if cmd.Long != "" {
		fmt.Fprintln(w, "DESCRIPTION")
		for _, line := range strings.Split(strings.TrimRight(cmd.Long, "\n"), "\n") {
			if line == "" {
				fmt.Fprintln(w)
			} else {
				fmt.Fprintf(w, "       %s\n", line)
			}
		}
		fmt.Fprintln(w)
	}

	// COMMANDS — non-hidden subcommands
	var subs []*cobra.Command
	for _, sub := range cmd.Commands() {
		if !sub.Hidden {
			subs = append(subs, sub)
		}
	}
	if len(subs) > 0 {
		fmt.Fprintln(w, "COMMANDS")
		for _, sub := range subs {
			fmt.Fprintf(w, "       %-24s %s\n", sub.Name(), sub.Short)
		}
		fmt.Fprintln(w)
	}

	// OPTIONS — flags specific to this command (non-inherited)
	flags := cmd.NonInheritedFlags()
	if flags.HasAvailableFlags() {
		fmt.Fprintln(w, "OPTIONS")
		for _, line := range strings.Split(strings.TrimRight(flags.FlagUsages(), "\n"), "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}

	// SEE ALSO
	var seeAlso []string
	if cmd.HasParent() && cmd.Parent().Name() != "" {
		seeAlso = append(seeAlso, cmd.Parent().CommandPath()+"(1)")
	}
	for _, sub := range subs {
		seeAlso = append(seeAlso, sub.CommandPath()+"(1)")
	}
	if len(seeAlso) > 0 {
		fmt.Fprintln(w, "SEE ALSO")
		fmt.Fprintf(w, "       %s\n\n", strings.Join(seeAlso, ", "))
	}

	fmt.Fprintln(w, sep)
	fmt.Fprintln(w, "Myr Admin CLI — myr man <command> to view any section.")
}
