package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "documentdbfuse",
	Short: "Mount any MongoDB-compatible database as a filesystem",
	Long: `DocumentDBFUSE mounts a MongoDB database via FUSE (Linux).
Browse collections with ls, read documents with cat, search with grep.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(buildMountCmd())
	rootCmd.AddCommand(buildUnmountCmd())
	rootCmd.AddCommand(buildVersionCmd())
}

func buildVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("documentdbfuse v0.1.0")
		},
	}
}
