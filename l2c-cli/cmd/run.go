package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/binodta/l2c/l2c-cli/internal/client"
	"github.com/spf13/cobra"
)

type Config struct {
	WorkerURL    string         `json:"worker_url,omitempty"`
	CustomDomain string         `json:"custom_domain,omitempty"`
	Token        string         `json:"token"`
	Tunnels      []TunnelConfig `json:"tunnels"`
}

type TunnelConfig struct {
	ID    string `json:"id"`
	Local string `json:"local"`
}

var configPath string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the tunnel client",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Fatalf("Error: %s not found. Run 'l2c setup' or create it manually.", configPath)
			}
			log.Fatalf("Error reading config: %v", err)
		}

		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Fatalf("Error parsing config: %v", err)
		}

		server := cfg.CustomDomain
		if server == "" {
			server = cfg.WorkerURL
		}

		if server == "" {
			log.Fatal("Error: 'worker_url' or 'custom_domain' must be specified in config.json")
		}

		if len(cfg.Tunnels) == 0 {
			log.Fatal("Error: No tunnels defined in config.json")
		}

		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

		var wg sync.WaitGroup
		clients := make([]*client.Client, 0)

		fmt.Printf("l2c-proxy starting...\n")
		fmt.Printf("Server: %s\n\n", server)

		for _, tc := range cfg.Tunnels {
			c := client.NewClient(server, tc.ID, tc.Local, cfg.Token)
			clients = append(clients, c)
			wg.Add(1)
			go func(cli *client.Client, id, local string) {
				defer wg.Done()
				if err := cli.Start(); err != nil {
					log.Printf("Tunnel [%s] error: %v", id, err)
				}
			}(c, tc.ID, tc.Local)
		}

		fmt.Printf("\nAll tunnels active! Press Ctrl+C to stop.\n")
		for _, tc := range cfg.Tunnels {
			fmt.Printf("- %s: https://%s/t/%s/ -> %s\n", tc.ID, server, tc.ID, tc.Local)
		}

		<-interrupt
		fmt.Println("\nShutting down all tunnels...")
		for _, c := range clients {
			c.Stop()
		}
		wg.Wait()
	},
}

func init() {
	home, _ := os.UserHomeDir()
	defaultConfig := ".l2c/config.json"
	if home != "" {
		defaultConfig = filepath.Join(home, ".l2c", "config.json")
	}
	runCmd.Flags().StringVarP(&configPath, "config", "c", defaultConfig, "Path to config file")
	rootCmd.AddCommand(runCmd)
}
