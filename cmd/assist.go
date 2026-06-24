package cmd

import (
	"fmt"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newAssistCmd manages Assist voice pipelines (conversation/STT/TTS chains).
// All endpoints are WebSocket-only. Running text through a pipeline is better
// done with `service call conversation.process`; the streaming
// assist_pipeline/run (audio) stays under `raw ws`.
func newAssistCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assist",
		Short: "Manage Assist voice pipelines",
		Example: `  hass-cli assist pipeline list
  hass-cli assist pipeline get
  hass-cli assist languages`,
	}
	cmd.AddCommand(newAssistPipelineCmd(f))

	cmd.AddCommand(&cobra.Command{
		Use:   "languages",
		Short: "List languages supported by available pipelines",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "assist_pipeline/language/list"}
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "devices",
		Short: "List devices bound to an Assist pipeline",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "assist_pipeline/device/list"}
		}),
	})

	return cmd
}

func newAssistPipelineCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "List and manage Assist pipelines",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List pipelines and the preferred one",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "assist_pipeline/pipeline/list"}
		}),
	})

	var getID string
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a pipeline (preferred one unless --id)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"type": "assist_pipeline/pipeline/get"}
			if getID != "" {
				body["pipeline_id"] = getID
			}
			return wsCall(f, cmd, body)
		},
	}
	getCmd.Flags().StringVar(&getID, "id", "", "Pipeline id (omit for preferred)")
	cmd.AddCommand(getCmd)

	var createData string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pipeline (--data with name/language/conversation_engine/...)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := parseDataObject(createData)
			if err != nil {
				return err
			}
			if body == nil {
				return fmt.Errorf(`--data is required (name, language, conversation_engine, stt_engine, tts_engine, ...)`)
			}
			body["type"] = "assist_pipeline/pipeline/create"
			return wsCall(f, cmd, body)
		},
	}
	createCmd.Flags().StringVar(&createData, "data", "", "Pipeline fields as JSON (or @file.json)")
	cmd.AddCommand(createCmd)

	var updateData string
	updateCmd := &cobra.Command{
		Use:   "update <pipeline_id>",
		Short: "Update a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := requireData(updateData)
			if err != nil {
				return err
			}
			body["type"] = "assist_pipeline/pipeline/update"
			body["pipeline_id"] = args[0]
			return wsCall(f, cmd, body)
		},
	}
	updateCmd.Flags().StringVar(&updateData, "data", "", "Fields to change as JSON (or @file.json)")
	cmd.AddCommand(updateCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "set-preferred <pipeline_id>",
		Short: "Mark a pipeline as the preferred default",
		Args:  cobra.ExactArgs(1),
		RunE: wsRun(f, func(args []string) map[string]any {
			return map[string]any{"type": "assist_pipeline/pipeline/set_preferred", "pipeline_id": args[0]}
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <pipeline_id>",
		Short: "Delete a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: wsRun(f, func(args []string) map[string]any {
			return map[string]any{"type": "assist_pipeline/pipeline/delete", "pipeline_id": args[0]}
		}),
	})

	return cmd
}
