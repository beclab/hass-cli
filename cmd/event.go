package cmd

import (
	"encoding/json"
	"os"
	"os/signal"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newEventCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event",
		Short: "Fire and watch events",
		Example: `  hass-cli event fire my_event --data '{"foo":"bar"}'
  hass-cli event watch state_changed`,
	}

	var dataJSON string
	fireCmd := &cobra.Command{
		Use:   "fire <event_type>",
		Short: "Fire a custom event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := parseDataObject(dataJSON)
			if err != nil {
				return err
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.FireEvent(cmd.Context(), args[0], data)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	fireCmd.Flags().StringVar(&dataJSON, "data", "", "Event data as a JSON object (or @file.json)")
	cmd.AddCommand(fireCmd)

	watchCmd := &cobra.Command{
		Use:   "watch [event_type]",
		Short: "Subscribe to events and stream them as NDJSON until interrupted",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer stop()

			payload := map[string]any{"type": "subscribe_events"}
			if len(args) == 1 {
				payload["event_type"] = args[0]
			}
			enc := json.NewEncoder(os.Stdout)
			err = c.Subscribe(ctx, payload, func(ev json.RawMessage) error {
				var v any
				if json.Unmarshal(ev, &v) == nil {
					return enc.Encode(v)
				}
				return nil
			})
			if ctx.Err() != nil {
				return nil
			}
			return err
		},
	}
	cmd.AddCommand(watchCmd)

	return cmd
}
