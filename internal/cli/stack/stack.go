package stack

import (
	"github.com/spf13/cobra"
)

// NewStackCmd creates the stack command group
func NewStackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Commands for operating on entire stacks",
		Long:  `Commands for operating on entire stacks of branches.`,
	}

	cmd.AddCommand(newStackInfoCmd())

	return cmd
}
