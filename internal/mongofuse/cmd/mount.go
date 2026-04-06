package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func buildMountCmd() *cobra.Command {
	var readOnly bool

	cmd := &cobra.Command{
		Use:   "mount <connection-string> <mount-point>",
		Short: "Mount a MongoDB database as a filesystem",
		Long: `Mount a MongoDB-compatible database at the specified mount point.

Examples:
  mongofuse mount "mongodb://localhost:27017" /mnt/db
  mongofuse mount "mongodb://user:pass@host:10260/?tls=true&tlsAllowInvalidCertificates=true&directConnection=true" /mnt/db
  mongofuse mount --read-only "mongodb://localhost:27017" /mnt/db`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			connString := args[0]
			mountPoint := args[1]

			fmt.Printf("Mounting %s at %s (read-only: %v)\n", connString, mountPoint, readOnly)

			// TODO: Initialize MongoDB client
			// TODO: Start FUSE/NFS server
			// TODO: Block until unmount signal

			return fmt.Errorf("mount not yet implemented")
		},
	}

	cmd.Flags().BoolVar(&readOnly, "read-only", false, "Mount in read-only mode")

	return cmd
}
