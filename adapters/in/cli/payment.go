// adapters/in/cli/payment.go — commandes Cobra pour les paiements
package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var paymentCmd = &cobra.Command{
	Use:   "payment",
	Short: "Manage payments",
	Long: `Commands to execute and query payments for 3D models.

Payments are recorded on the blockchain and form an immutable audit trail
of transactions between buyers and sellers of models.`,
}

var paymentPayCmd = &cobra.Command{
	Use:   "pay <from> <to> <modelID> <amount>",
	Short: "Execute a payment for a model",
	Long: `Record a payment on the blockchain between two identities for
access to a 3D model.

Arguments:
  from     Buyer identity identifier
  to       Seller identity identifier
  modelID  Identifier of the model being purchased
  amount   Amount (decimal, e.g. 9.99)

Example:
  myr payment pay alice bob abc123 9.99`,
	Args: cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		amount, err := strconv.ParseFloat(args[3], 64)
		if err != nil {
			return fmt.Errorf("montant invalide : %w", err)
		}

		p, err := paymentSvc.Pay(args[0], args[1], args[2], amount)
		if err != nil {
			return err
		}
		fmt.Printf("Paiement effectué : id=%s  %.2f  %s → %s  modèle=%s\n",
			p.ID, p.Amount, p.From, p.To, p.ModelID)
		return nil
	},
}

var paymentHistoryCmd = &cobra.Command{
	Use:   "history <identityID>",
	Short: "Show payment history for an identity",
	Long: `List all transactions (incoming and outgoing) associated with a
given identity, in chronological order.

Example:
  myr payment history alice`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		history, err := paymentSvc.GetHistory(args[0])
		if err != nil {
			return err
		}
		for _, p := range history {
			fmt.Printf("%s  %s → %s  %.2f  modèle=%s\n",
				p.ID, p.From, p.To, p.Amount, p.ModelID)
		}
		return nil
	},
}

func init() {
	paymentCmd.AddCommand(paymentPayCmd, paymentHistoryCmd)
}
