package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/router"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List and inspect models",
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List roles and models",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := config.LoadMergedManifest()
		if err != nil {
			return err
		}
		for _, role := range []string{"worker", "router", "escalated"} {
			md, err := m.ResolveRole(role)
			if err != nil {
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s %s\n", role, md.ID, md.Provider, md.Capabilities.ToolProtocol)
		}
		return nil
	},
}

var modelsShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show one model",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := config.LoadMergedManifest()
		if err != nil {
			return err
		}
		md, err := m.ResolveModel(args[0])
		if err != nil {
			return err
		}
		rates := "unset"
		if md.InputPerM != 0 || md.OutputPerM != 0 {
			rates = fmt.Sprintf("in=%.4f out=%.4f", md.InputPerM, md.OutputPerM)
		}
		fmt.Printf("%s provider=%s model=%s protocol=%s rates=%s\n", md.ID, md.Provider, md.Model, md.Capabilities.ToolProtocol, rates)
		return nil
	},
}

var modelsProbeCmd = &cobra.Command{
	Use:   "probe [id]",
	Short: "Probe a model endpoint and write a catalog snippet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := config.LoadMergedManifest()
		if err != nil {
			return err
		}
		md, err := m.ResolveModel(args[0])
		if err != nil {
			return err
		}
		pd, err := m.ProviderForModel(md)
		if err != nil {
			return err
		}
		key, _ := config.ResolveProviderKey(pd.APIKeyEnv, md.Provider)
		sn, err := router.ProbeChat(context.Background(), pd.BaseURL, key, md.ID)
		if err != nil {
			return err
		}
		if err := router.WriteSnippet(sn); err != nil {
			return err
		}
		fmt.Printf("probe %s structured_output=%s tool_protocol=%s\n%s\n", sn.ModelID, sn.StructuredOutput, sn.ToolProtocol, sn.Note)
		return nil
	},
}

func init() {
	modelsCmd.AddCommand(modelsListCmd, modelsShowCmd, modelsProbeCmd)
	rootCmd.AddCommand(modelsCmd)
}
