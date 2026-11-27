package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/beclab/hass-cli/internal/cmdutil"
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
		Long: `Manage automations, scripts, and scenes. Each subcommand takes a domain
(automation|script|scene) and supports list/get/save/delete/reload/trigger,
and stop for scripts and automations.
"save" takes the config as --data: inline JSON, or @file with YAML or JSON.`,
		Example: `  hass-cli workflow script save color_cycle --data @color_cycle.yaml
  hass-cli workflow script trigger script.color_cycle
  hass-cli workflow script stop script.color_cycle
  hass-cli workflow automation list`,
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
	refuseUnknownVerbs(cmd)

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
			return renderValueCols(f, filtered, colsState)
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

	var data string
	saveCmd := &cobra.Command{
		Use:   "save <id>",
		Short: fmt.Sprintf("Create or update a %s by id", domain),
		Long:  saveLong(domain),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := checkConfigID(domain, id); err != nil {
				return err
			}
			if data == "" {
				return fmt.Errorf("--data is required: the %s config as inline JSON, or @file with YAML or JSON", domain)
			}
			raw, err := readDataArg(data)
			if err != nil {
				return err
			}
			var body map[string]any
			if err := yaml.Unmarshal(raw, &body); err != nil || len(body) == 0 {
				return fmt.Errorf("--data must be a non-empty %s config object (YAML or JSON): %v", domain, err)
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			res, err := c.REST(cmd.Context(), "POST", configPath(domain, id), body)
			if err != nil {
				return err
			}
			if f.Output != "" && f.Output != "table" {
				return renderRaw(f, res)
			}
			r, err := f.Renderer()
			if err != nil {
				return err
			}
			switch domain {
			case "script":
				_, err = fmt.Fprintf(r.Out, "Saved script.%s. Run it: hass-cli workflow script trigger script.%s\n", id, id)
			default:
				_, err = fmt.Fprintf(r.Out, "Saved %s %s. Its entity id comes from its alias: hass-cli workflow %s list\n", domain, id, domain)
			}
			return err
		},
	}
	saveCmd.Flags().StringVar(&data, "data", "", "The config: inline JSON, or @file with YAML or JSON")
	saveCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return fmt.Errorf("%w\nsave takes the config as --data '<json>' or --data @file.yaml", err)
	})
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

	// Home Assistant runs a script with script.turn_on and a scene with
	// scene.turn_on; only automations have a trigger service.
	runService := map[string]string{"automation": "trigger", "script": "turn_on", "scene": "turn_on"}[domain]
	cmd.AddCommand(&cobra.Command{
		Use:   "trigger <entity_id>",
		Short: fmt.Sprintf("Run a %s now (%s.%s)", domain, domain, runService),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.CallService(cmd.Context(), domain, runService, map[string]any{"entity_id": args[0]})
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	if domain != "scene" {
		cmd.AddCommand(&cobra.Command{
			Use:   "stop <entity_id>",
			Short: fmt.Sprintf("Stop a running %s", domain),
			Long:  stopLong(domain),
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				c, err := f.Client()
				if err != nil {
					return err
				}
				defer c.Close()
				target := map[string]any{"entity_id": args[0]}
				if domain == "script" {
					raw, err := c.CallService(cmd.Context(), "script", "turn_off", target)
					if err != nil {
						return err
					}
					return renderRaw(f, raw)
				}
				if _, err := c.CallService(cmd.Context(), "automation", "turn_off", map[string]any{"entity_id": args[0], "stop_actions": true}); err != nil {
					return err
				}
				raw, err := c.CallService(cmd.Context(), "automation", "turn_on", target)
				if err != nil {
					return fmt.Errorf("stopped %s but could not enable it again: %w", args[0], err)
				}
				return renderRaw(f, raw)
			},
		})
	}

	return cmd
}

func saveLong(domain string) string {
	switch domain {
	case "script":
		return `Create or update a script. <id> is the bare id: saving color_cycle makes the
entity script.color_cycle. Lowercase letters, digits and underscores only.
Home Assistant reloads scripts on save.`
	case "automation":
		return `Create or update an automation. <id> is the config id (any string; the UI uses
32 hex characters). The entity id comes from the alias, not the id.
Home Assistant reloads automations on save.`
	}
	return `Create or update a scene. <id> is the config id; the entity id comes from
the scene's name.`
}

func stopLong(domain string) string {
	if domain == "script" {
		return "Stop a running script (script.turn_off). The script stays defined and can run again."
	}
	return `Stop an automation's running actions and leave it enabled: automation.turn_off
with stop_actions, then automation.turn_on.`
}

var scriptIDRe = regexp.MustCompile(`^[a-z0-9_]+$`)

// checkConfigID refuses ids Home Assistant would reject, before the request.
func checkConfigID(domain, id string) error {
	if bare, ok := strings.CutPrefix(id, domain+"."); ok {
		return fmt.Errorf("save takes the bare id: %s (the entity becomes %s.%s)", bare, domain, bare)
	}
	if domain == "script" && !scriptIDRe.MatchString(id) {
		suggestion := slugify(id)
		if suggestion == "" {
			suggestion = "an English name such as color_cycle"
		}
		return fmt.Errorf("%q is not a script id; use %s (lowercase letters, digits, underscores)", id, suggestion)
	}
	return nil
}

func slugify(s string) string {
	var b strings.Builder
	underscore := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			underscore = false
			continue
		}
		if !underscore && b.Len() > 0 {
			b.WriteByte('_')
			underscore = true
		}
	}
	return strings.TrimSuffix(b.String(), "_")
}

func configPath(domain, id string) string {
	return fmt.Sprintf("config/%s/config/%s", domain, id)
}
