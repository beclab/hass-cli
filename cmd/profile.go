package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"golang.org/x/term"

	"github.com/beclab/hass-cli/internal/client"
	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/beclab/hass-cli/internal/config"
	"github.com/beclab/hass-cli/internal/keychain"
	"github.com/beclab/hass-cli/internal/profile"
)

func newProfileCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage hass-cli profiles (server + token)",
		Long: `Manage hass-cli profiles. A profile bundles a Home Assistant server URL with
the long-lived access token used to reach it.

Tokens are stored in the OS keychain (service "hass-cli", account = profile
name): macOS Keychain on darwin, an AES-256-GCM file under ~/.local/share on
linux, and DPAPI under HKCU\Software\HassCli on windows. The profile index
(server, timeout, no secrets) lives in profiles.json under the config dir.`,
	}
	for _, sub := range []*cobra.Command{
		newProfileLoginCmd(),
		newProfileListCmd(),
		newProfileUseCmd(),
		newProfileRemoveCmd(),
		newProfileCurrentCmd(),
		newProfileShowCmd(),
	} {
		sub.SilenceUsage = true
		cmd.AddCommand(sub)
	}
	return cmd
}

// loginOptions carries the inputs for `profile login`.
type loginOptions struct {
	name        string
	server      string
	token       string
	tokenStdin  bool
	insecure    bool
	timeout     int
	force       bool
	noSwitch    bool
	interactive bool // when true, prompt for missing server/token on a TTY
}

func newProfileLoginCmd() *cobra.Command {
	o := &loginOptions{}
	cmd := &cobra.Command{
		Use:   "login [name]",
		Short: "Validate a server + token and save it as a profile",
		Long: `Authenticate to a Home Assistant instance by validating a long-lived access
token against /api/config, then store it in the OS keychain under a named
profile and make it the current profile.

The token is read from --token, from stdin with --token-stdin, or prompted
(hidden) on a terminal. Re-running login against a profile that still has a
valid token is rejected unless --force is passed.`,
		Example: `  hass-cli profile login home --server http://homeassistant.local:8123
  printf '%s' "$TOKEN" | hass-cli profile login home --server https://ha.example.com --token-stdin`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				o.name = args[0]
			}
			o.interactive = true
			if _, err := runProfileLogin(cmd.Context(), o); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "\nNext steps:")
			fmt.Fprintln(out, "  hass-cli ping              # verify connectivity")
			fmt.Fprintln(out, "  hass-cli skill list        # browse bundled agent skills")
			return nil
		},
	}
	cmd.Flags().StringVarP(&o.server, "server", "s", "", "Home Assistant URL (e.g. http://homeassistant.local:8123)")
	cmd.Flags().StringVar(&o.token, "token", "", "Long-lived access token (omit to read from stdin or prompt)")
	cmd.Flags().BoolVar(&o.tokenStdin, "token-stdin", false, "Read the token from stdin instead of prompting")
	cmd.Flags().BoolVar(&o.insecure, "insecure", false, "Skip TLS certificate verification for this profile")
	cmd.Flags().IntVar(&o.timeout, "timeout", 0, "Request timeout in seconds (default 10)")
	cmd.Flags().BoolVar(&o.force, "force", false, "Overwrite an existing profile that still has a valid token")
	cmd.Flags().BoolVar(&o.noSwitch, "no-switch", false, "Do not make this profile current after login")
	return cmd
}

// runProfileLogin validates server+token, stores the token in the keychain,
// and upserts the profile. Returns the persisted entry. It performs the
// interactive URL + hidden-token prompts when run on a terminal, so
// `profile login` doubles as the first-run setup wizard.
func runProfileLogin(ctx context.Context, o *loginOptions) (*profile.Entry, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if o.server == "" && o.interactive && isTTY() {
		v, err := promptLine("Home Assistant URL (e.g. http://homeassistant.local:8123): ")
		if err != nil {
			return nil, err
		}
		o.server = v
	}
	o.server = strings.TrimSpace(o.server)
	if o.server == "" {
		return nil, errors.New("--server is required")
	}

	if o.name == "" {
		o.name = "default"
	}

	token, err := readToken(o)
	if err != nil {
		return nil, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("token is empty")
	}

	idx, err := profile.Load()
	if err != nil {
		return nil, err
	}
	store := profile.NewTokenStore()

	if !o.force {
		if existing := idx.Find(o.name); existing != nil {
			if status := profile.Status(store, o.name, time.Now()); status == "logged-in" {
				return nil, fmt.Errorf(
					"profile %q already has a valid token.\nto replace it, re-run with --force or run: hass-cli profile remove %s",
					o.name, o.name)
			}
		}
	}

	timeout := o.timeout
	if timeout <= 0 {
		timeout = 10
	}
	cfg := &config.Config{
		Server:         o.server,
		Token:          token,
		Insecure:       o.insecure,
		TimeoutSeconds: timeout,
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	c := client.New(cfg)
	defer c.Close()
	raw, err := c.Config(ctx)
	if err != nil {
		return nil, fmt.Errorf("validate token against %s: %w", o.server, err)
	}

	entry := profile.Entry{
		Name:         o.name,
		Server:       o.server,
		Insecure:     o.insecure,
		Timeout:      o.timeout,
		InstanceName: gjson.GetBytes(raw, "location_name").String(),
		HAVersion:    gjson.GetBytes(raw, "version").String(),
	}

	if err := store.Set(o.name, token); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}
	persisted := idx.Upsert(entry)
	switched := false
	if !o.noSwitch || idx.CurrentProfile == "" {
		if idx.CurrentProfile != persisted.Name {
			if _, err := idx.SetCurrent(persisted.Name); err != nil {
				return nil, err
			}
			switched = true
		}
	}
	if err := profile.Save(idx); err != nil {
		return nil, fmt.Errorf("save profiles: %w", err)
	}

	who := persisted.InstanceName
	if who == "" {
		who = o.server
	}
	fmt.Printf("logged in to %s", who)
	if persisted.HAVersion != "" {
		fmt.Printf(" (Home Assistant %s)", persisted.HAVersion)
	}
	fmt.Printf(" — profile %q\n", persisted.Name)
	if switched {
		fmt.Printf("current profile is now %q\n", persisted.Name)
	}
	fmt.Printf("token stored via %s (service %q, account %q)\n",
		keychain.Backend(keychain.HassCliService), keychain.HassCliService, persisted.Name)
	return persisted, nil
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles with login status and current marker",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			idx, err := profile.Load()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(idx.Profiles) == 0 {
				fmt.Fprintln(out, "no profiles configured.")
				fmt.Fprintln(out, "run `hass-cli profile login <name> --server <url>` to add one.")
				return nil
			}
			store := profile.NewTokenStore()
			current := idx.Current()
			now := time.Now()
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  \tNAME\tSERVER\tSTATUS\tVERSION")
			for i := range idx.Profiles {
				p := &idx.Profiles[i]
				marker := " "
				if current != nil && current.Name == p.Name {
					marker = "*"
				}
				version := p.HAVersion
				if version == "" {
					version = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					marker, p.Name, p.Server, profile.Status(store, p.Name, now), version)
			}
			return w.Flush()
		},
	}
}

func newProfileUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name|->",
		Short: "Switch the current profile (use `-` to switch back)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := profile.Load()
			if err != nil {
				return err
			}
			target, err := idx.SetCurrent(args[0])
			if err != nil {
				return err
			}
			if err := profile.Save(idx); err != nil {
				return fmt.Errorf("save profiles: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "switched to profile %q (%s)\n", target.Name, target.Server)
			return nil
		},
	}
}

func newProfileRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a profile and its stored token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := profile.Load()
			if err != nil {
				return err
			}
			removed, ok := idx.Remove(args[0])
			if !ok {
				return fmt.Errorf("profile %q not found", args[0])
			}
			if err := profile.Save(idx); err != nil {
				return fmt.Errorf("save profiles: %w", err)
			}
			out := cmd.OutOrStdout()
			store := profile.NewTokenStore()
			if err := store.Delete(removed.Name); err != nil && !errors.Is(err, profile.ErrTokenNotFound) {
				fmt.Fprintf(out, "warning: failed to clear stored token for %q: %v\n", removed.Name, err)
			}
			fmt.Fprintf(out, "removed profile %q (%s)\n", removed.Name, removed.Server)
			if idx.CurrentProfile != "" {
				fmt.Fprintf(out, "current profile is now %q\n", idx.CurrentProfile)
			} else if len(idx.Profiles) == 0 {
				fmt.Fprintln(out, "no profiles remain.")
				if err := keychain.PurgeService(keychain.HassCliService); err != nil {
					fmt.Fprintf(out, "warning: failed to purge keychain storage: %v\n", err)
				}
			}
			return nil
		},
	}
}

func newProfileCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show the current profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			idx, err := profile.Load()
			if err != nil {
				return err
			}
			cur := idx.Current()
			if cur == nil {
				return errors.New("no current profile; run `hass-cli profile login`")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n",
				cur.Name, cur.Server, profile.Status(profile.NewTokenStore(), cur.Name, time.Now()))
			return nil
		},
	}
}

func newProfileShowCmd() *cobra.Command {
	var validate bool
	cmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show a profile's details (masked token)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := profile.Load()
			if err != nil {
				return err
			}
			var e *profile.Entry
			if len(args) == 1 {
				e = idx.Find(args[0])
			} else {
				e = idx.Current()
			}
			if e == nil {
				return errors.New("profile not found; run `hass-cli profile list`")
			}
			store := profile.NewTokenStore()
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "name:     %s\n", e.Name)
			fmt.Fprintf(out, "server:   %s\n", e.Server)
			fmt.Fprintf(out, "insecure: %t\n", e.Insecure)
			if e.InstanceName != "" {
				fmt.Fprintf(out, "instance: %s\n", e.InstanceName)
			}
			if e.HAVersion != "" {
				fmt.Fprintf(out, "version:  %s\n", e.HAVersion)
			}
			tok, terr := store.Get(e.Name)
			if terr != nil {
				fmt.Fprintf(out, "token:    <none stored>\n")
			} else {
				fmt.Fprintf(out, "token:    %s\n", maskToken(tok))
			}
			fmt.Fprintf(out, "status:   %s\n", profile.Status(store, e.Name, time.Now()))

			if validate {
				cfg := &config.Config{Server: e.Server, Token: tok, Insecure: e.Insecure, TimeoutSeconds: 10}
				if err := cfg.Validate(); err != nil {
					return err
				}
				c := client.New(cfg)
				defer c.Close()
				if _, err := c.Config(cmd.Context()); err != nil {
					fmt.Fprintf(out, "live:     unreachable/invalid (%s)\n", client.FriendlyMessage(err))
					return nil
				}
				fmt.Fprintf(out, "live:     ok\n")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&validate, "validate", false, "Check the profile against the live instance")
	return cmd
}

// readToken obtains the token from --token, stdin, or an interactive prompt.
func readToken(o *loginOptions) (string, error) {
	if o.token != "" {
		return o.token, nil
	}
	if o.tokenStdin {
		return readSingleLine(os.Stdin)
	}
	if o.interactive && isTTY() {
		fmt.Print("Paste long-lived access token: ")
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("read token: %w", err)
		}
		return string(b), nil
	}
	return "", errors.New("no token provided: pass --token, --token-stdin, or run on a terminal")
}

func readSingleLine(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	line, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func promptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	br := bufio.NewReader(os.Stdin)
	line, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func isTTY() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

func maskToken(tok string) string {
	if len(tok) <= 12 {
		return "****"
	}
	return tok[:6] + "..." + tok[len(tok)-4:]
}
