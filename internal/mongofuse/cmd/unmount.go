package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func buildUnmountCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unmount <mount-point>",
		Short: "Unmount a MongoFUSE filesystem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mountPoint := args[0]
			fmt.Printf("Unmounting %s\n", mountPoint)
			// TODO: Implement unmount
			return fmt.Errorf("unmount not yet implemented")
		},
	}
}
