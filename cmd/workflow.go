package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"
)

// newWorkflowCmds builds automation/script/scene command groups. Config CRUD
// uses the REST config editor endpoints; reload/trigger use call_service.
func newWorkflowCmds(f *cmdutil.Factory) *cobra.Command {
	parent := &cobra.Command{
		Use:   "workflow",
		Short: "Manage automations, scripts, and scenes",
	}
	for _, domain := range []string{"automation", "script", "scene"} {
		parent.AddCommand(newWorkflowCmd(f, domain))
	}
	return parent
}

func newWorkflowCmd(f *cmdutil.Factory, domain string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   domain,
		Short: fmt.Sprintf("Operate on %ss", domain),
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List %s entities", domain),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.States(cmd.Context())
			if err != nil {
				return err
			}
			var filtered []any
			gjson.ParseBytes(raw).ForEach(func(_, v gjson.Result) bool {
				if strings.HasPrefix(v.Get("entity_id").String(), domain+".") {
					var item any
					_ = json.Unmarshal([]byte(v.Raw), &item)
					filtered = append(filtered, item)
				}
				return true
			})
			return renderValue(f, filtered)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <id>",
		Short: fmt.Sprintf("Get a %s configuration by id", domain),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "GET", configPath(domain, args[0]), nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	var file string
	saveCmd := &cobra.Command{
		Use:   "save <id>",
		Short: fmt.Sprintf("Create or update a %s from a YAML/JSON file", domain),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			var body any
			if err := yaml.Unmarshal(data, &body); err != nil {
				return fmt.Errorf("parse %s: %w", file, err)
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "POST", configPath(domain, args[0]), body)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	saveCmd.Flags().StringVar(&file, "file", "", "Config file (YAML or JSON)")
	cmd.AddCommand(saveCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <id>",
		Short: fmt.Sprintf("Delete a %s by id", domain),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "DELETE", configPath(domain, args[0]), nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "reload",
		Short: fmt.Sprintf("Reload %ss", domain),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.CallService(cmd.Context(), domain, "reload", nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "trigger <entity_id>",
		Short: fmt.Sprintf("Trigger a %s", domain),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			service := "trigger"
			if domain == "scene" {
				service = "turn_on"
			}
			raw, err := c.CallService(cmd.Context(), domain, service, map[string]any{"entity_id": args[0]})
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	return cmd
}

func configPath(domain, id string) string {
	return fmt.Sprintf("config/%s/config/%s", domain, id)
}
