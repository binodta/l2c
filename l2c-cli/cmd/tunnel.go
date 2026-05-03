package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// ── Validation ────────────────────────────────────────────────────────────────

// validTunnelID matches lowercase alphanumeric segments separated by hyphens,
// mirroring DNS label rules (no leading/trailing hyphens, max 63 chars).
var validTunnelID = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`)

// validateTunnelID returns a descriptive error when id does not meet the rules.
func validateTunnelID(id string) error {
	if id == "" {
		return fmt.Errorf("tunnel ID must not be empty")
	}
	if id == "connect" {
		return fmt.Errorf("tunnel ID 'connect' is reserved")
	}
	if len(id) > 63 {
		return fmt.Errorf("tunnel ID must be 63 characters or fewer (got %d)", len(id))
	}
	if !validTunnelID.MatchString(id) {
		return fmt.Errorf(
			"tunnel ID %q is invalid — only lowercase letters, digits, and hyphens are allowed; "+
				"must start and end with a letter or digit",
			id,
		)
	}
	return nil
}

// validateLocalURL returns an error when addr is not a valid http/https URL.
// It also normalises the value in-place (trims trailing slash).
func validateLocalURL(addr string) (string, error) {
	if addr == "" {
		return "", fmt.Errorf("--local must not be empty")
	}
	u, err := url.Parse(addr)
	if err != nil {
		return "", fmt.Errorf("--local %q is not a valid URL: %w", addr, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("--local %q must use http:// or https:// scheme", addr)
	}
	if u.Host == "" {
		return "", fmt.Errorf("--local %q must include a host (e.g. http://localhost:3000)", addr)
	}
	// Normalise: strip a trailing slash from the path so routes are consistent.
	normalised := strings.TrimRight(addr, "/")
	return normalised, nil
}

// ── Config I/O ────────────────────────────────────────────────────────────────

// loadConfig reads and parses the config file at path.
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config at %s: %w", path, err)
	}
	return &cfg, nil
}

// saveConfig writes cfg to path atomically: it first writes to a sibling .tmp
// file, then renames it into place. This prevents a crash mid-write from
// leaving the config in a corrupt state.
func saveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	// Write to a temp file in the same directory so Rename is atomic on Linux.
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file for atomic write: %w", err)
	}
	tmpName := tmp.Name()

	// Always clean up the temp file on any failure path.
	success := false
	defer func() {
		if !success {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write config: %w", err)
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to flush config to disk: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to save config (rename failed): %w", err)
	}

	success = true
	return nil
}

// defaultConfigPath returns the canonical config file path (~/.l2c/config.json).
func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".l2c", "config.json")
	}
	return ".l2c/config.json"
}

// ── tunnel (parent) ──────────────────────────────────────────────────────────

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage tunnels in the config",
}

// ── tunnel add ───────────────────────────────────────────────────────────────

var (
	addTunnelID    string
	addTunnelLocal string
	addConfigPath  string
	addRewriteHost bool
)

var tunnelAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new tunnel to the config",
	Example: `  l2c tunnel add --id api --local http://localhost:3000
  l2c tunnel add --id web --local http://localhost:8080 --config /path/to/config.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate ID format.
		if err := validateTunnelID(addTunnelID); err != nil {
			return err
		}

		// Validate and normalise local URL.
		normLocal, err := validateLocalURL(addTunnelLocal)
		if err != nil {
			return err
		}

		cfg, err := loadConfig(addConfigPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("config not found at %s — run 'l2c setup' first", addConfigPath)
			}
			return fmt.Errorf("error reading config: %w", err)
		}

		// Reject duplicate IDs.
		for _, t := range cfg.Tunnels {
			if t.ID == addTunnelID {
				return fmt.Errorf(
					"tunnel with id %q already exists — use 'l2c tunnel list' to see current tunnels",
					addTunnelID,
				)
			}
		}

		cfg.Tunnels = append(cfg.Tunnels, TunnelConfig{
			ID:          addTunnelID,
			Local:       normLocal,
			RewriteHost: addRewriteHost,
		})

		if err := saveConfig(addConfigPath, cfg); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}

		server := cfg.CustomDomain
		if server == "" {
			server = cfg.WorkerURL
		}
		if server == "" {
			return fmt.Errorf("neither worker_url nor custom_domain is configured in config; please run 'l2c setup'")
		}

		fmt.Printf("✓ Tunnel %q added → %s\n", addTunnelID, normLocal)
		fmt.Printf("  Public URL: https://%s/%s/\n", server, addTunnelID)
		return nil
	},
}

// ── tunnel list ──────────────────────────────────────────────────────────────

var listConfigPath string

var tunnelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tunnels in the config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(listConfigPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("config not found at %s — run 'l2c setup' first", listConfigPath)
			}
			return fmt.Errorf("error reading config: %w", err)
		}

		if len(cfg.Tunnels) == 0 {
			fmt.Println("No tunnels configured. Use 'l2c tunnel add' to add one.")
			return nil
		}

		server := cfg.CustomDomain
		if server == "" {
			server = cfg.WorkerURL
		}
		if server == "" {
			return fmt.Errorf("neither worker_url nor custom_domain is configured in config; please run 'l2c setup'")
		}

		fmt.Printf("Server: %s\n\n", server)
		fmt.Printf("%-20s  %-30s  %s\n", "ID", "Local", "Public URL")
		fmt.Printf("%-20s  %-30s  %s\n",
			"--------------------",
			"------------------------------",
			"----------",
		)
		for _, t := range cfg.Tunnels {
			fmt.Printf("%-20s  %-30s  https://%s/%s/\n", t.ID, t.Local, server, t.ID)
		}
		return nil
	},
}

// ── tunnel remove ────────────────────────────────────────────────────────────

var (
	removeTunnelID   string
	removeConfigPath string
	removeForce      bool
)

var tunnelRemoveCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a tunnel from the config",
	Example: `  l2c tunnel remove --id api
  l2c tunnel remove --id api --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if removeTunnelID == "" {
			return fmt.Errorf("--id is required")
		}

		cfg, err := loadConfig(removeConfigPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("config not found at %s — run 'l2c setup' first", removeConfigPath)
			}
			return fmt.Errorf("error reading config: %w", err)
		}

		// Find the tunnel first so we can show its details before confirming.
		var target *TunnelConfig
		for i := range cfg.Tunnels {
			if cfg.Tunnels[i].ID == removeTunnelID {
				target = &cfg.Tunnels[i]
				break
			}
		}
		if target == nil {
			return fmt.Errorf(
				"tunnel %q not found — use 'l2c tunnel list' to see current tunnels",
				removeTunnelID,
			)
		}

		// Confirmation prompt unless --force is set.
		if !removeForce {
			fmt.Printf("Remove tunnel %q (→ %s)? [y/N] ", target.ID, target.Local)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.ToLower(strings.TrimSpace(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Build a new slice without the removed entry (no aliasing).
		filtered := make([]TunnelConfig, 0, len(cfg.Tunnels)-1)
		for _, t := range cfg.Tunnels {
			if t.ID != removeTunnelID {
				filtered = append(filtered, t)
			}
		}
		cfg.Tunnels = filtered

		if err := saveConfig(removeConfigPath, cfg); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}

		fmt.Printf("✓ Tunnel %q removed.\n", removeTunnelID)
		return nil
	},
}

// ── init ─────────────────────────────────────────────────────────────────────

func init() {
	cfgDefault := defaultConfigPath()

	// tunnel add
	tunnelAddCmd.Flags().StringVar(&addTunnelID, "id", "", "Unique tunnel ID — lowercase letters, digits, hyphens (e.g. my-api)")
	tunnelAddCmd.Flags().StringVar(&addTunnelLocal, "local", "", "Local address to forward to (e.g. http://localhost:3000)")
	tunnelAddCmd.Flags().BoolVar(&addRewriteHost, "rewrite-host", false, "Rewrite the Host header to match the local address")
	tunnelAddCmd.Flags().StringVarP(&addConfigPath, "config", "c", cfgDefault, "Path to config file")

	// tunnel list
	tunnelListCmd.Flags().StringVarP(&listConfigPath, "config", "c", cfgDefault, "Path to config file")

	// tunnel remove
	tunnelRemoveCmd.Flags().StringVar(&removeTunnelID, "id", "", "ID of the tunnel to remove")
	tunnelRemoveCmd.Flags().StringVarP(&removeConfigPath, "config", "c", cfgDefault, "Path to config file")
	tunnelRemoveCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Skip confirmation prompt")

	// assemble tree
	tunnelCmd.AddCommand(tunnelAddCmd)
	tunnelCmd.AddCommand(tunnelListCmd)
	tunnelCmd.AddCommand(tunnelRemoveCmd)
	rootCmd.AddCommand(tunnelCmd)
}
