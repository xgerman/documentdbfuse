package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func buildUnmountCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unmount <mount-point>",
		Short: "Unmount a DocumentDBFUSE filesystem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mountPoint := args[0]
			out, err := exec.Command("fusermount", "-u", mountPoint).CombinedOutput()
			if err != nil {
				return fmt.Errorf("unmount failed: %w\n%s", err, out)
			}
			fmt.Printf("Unmounted %s\n", mountPoint)
			return nil
		},
	}
}
