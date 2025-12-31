package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"noldermd/internal/server"
)

func main() {
	rootCmd := newRootCmd(server.Run)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd(runServer func(server.Config) error) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "noldermd",
		Short: "NolderMD - Markdown notes server",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the NolderMD server",
		RunE: func(cmd *cobra.Command, args []string) error {
			notesDir, err := cmd.Flags().GetString("notes-dir")
			if err != nil {
				return err
			}
			port, err := cmd.Flags().GetInt("port")
			if err != nil {
				return err
			}
			logLevel, err := cmd.Flags().GetString("log-level")
			if err != nil {
				return err
			}

			cfg := server.Config{
				NotesDir: notesDir,
				Port:     port,
				LogLevel: logLevel,
			}

			fmt.Printf("NolderMD listening on http://localhost:%d (notes: %s)\n", port, notesDir)
			return runServer(cfg)
		},
	}

	serveCmd.Flags().String("notes-dir", "./Notes", "Path to the notes directory")
	serveCmd.Flags().Int("port", 8080, "Port to listen on")
	serveCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")

	rootCmd.AddCommand(serveCmd)

	return rootCmd
}
