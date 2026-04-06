package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/xgerman/mongofuse/internal/mongofuse/db"
	"github.com/xgerman/mongofuse/internal/mongofuse/fs"
	fusemod "github.com/xgerman/mongofuse/internal/mongofuse/fuse"
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

			ctx := context.Background()

			// Ensure mount point directory exists.
			if err := os.MkdirAll(mountPoint, 0755); err != nil {
				return fmt.Errorf("failed to create mount point: %w", err)
			}

			client, err := db.NewClient(ctx, connString)
			if err != nil {
				return fmt.Errorf("failed to connect to MongoDB: %w", err)
			}

			ops := fs.NewOperations(client)

			var mountOpts []string
			if readOnly {
				mountOpts = append(mountOpts, "ro")
			}

			server, err := fusemod.Server(mountPoint, ops, mountOpts...)
			if err != nil {
				_ = client.Close(ctx)
				return fmt.Errorf("failed to start FUSE server: %w", err)
			}

			// fs.Mount already starts serving in the background.
			// Wait for SIGINT/SIGTERM to trigger clean shutdown.
			fmt.Printf("MongoFUSE mounted at %s\n", mountPoint)

			// Wait for SIGINT/SIGTERM to trigger clean shutdown.
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			fmt.Fprintln(os.Stderr, "\nUnmounting...")
			if err := server.Unmount(); err != nil {
				fmt.Fprintf(os.Stderr, "unmount error: %v\n", err)
			}
			if err := client.Close(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "client close error: %v\n", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&readOnly, "read-only", false, "Mount in read-only mode")

	return cmd
}
