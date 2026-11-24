package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Store provider credentials",
}

var authLoginCmd = &cobra.Command{
	Use:   "login [provider]",
	Short: "Save a provider key in the OS keyring",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, _ := cmd.Flags().GetString("key")
		if key == "" {
			fmt.Fprint(os.Stderr, "key: ")
			sc := bufio.NewScanner(os.Stdin)
			if sc.Scan() {
				key = strings.TrimSpace(sc.Text())
			}
		}
		id := args[0]
		if err := config.StoreProviderKey(id, key); err != nil {
			return err
		}
		m, err := config.LoadMergedManifest()
		if err != nil {
			return err
		}
		base, envName := "", ""
		if p, ok := m.Providers[id]; ok {
			base = p.BaseURL
			envName = p.APIKeyEnv
		}
		if err := config.RememberProvider(id, base, envName); err != nil {
			return err
		}
		fmt.Printf("stored key for %s\n", id)
		return nil
	},
}

func init() {
	authLoginCmd.Flags().String("key", "", "Provider key (prefer stdin; never written to the catalog)")
	authCmd.AddCommand(authLoginCmd)
	rootCmd.AddCommand(authCmd)
}
