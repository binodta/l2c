package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/binodta/l2c/l2c-cli/internal/client"
)

type Config struct {
	Server  string         `json:"server"`
	Token   string         `json:"token"`
	Tunnels []TunnelConfig `json:"tunnels"`
}

type TunnelConfig struct {
	ID    string `json:"id"`
	Local string `json:"local"`
}

func main() {
	configPath := flag.String("config", "config.json", "Path to config file (JSON)")
	flag.Parse()

	data, err := os.ReadFile(*configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("Error: %s not found. Please create it or use -config to specify a file.", *configPath)
		}
		log.Fatalf("Error reading config: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Error parsing config: %v", err)
	}

	if cfg.Server == "" {
		log.Fatal("Error: 'server' address must be specified in config.json")
	}

	if len(cfg.Tunnels) == 0 {
		log.Fatal("Error: No tunnels defined in config.json")
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	clients := make([]*client.Client, 0)

	fmt.Printf("l2c-proxy starting...\n")
	fmt.Printf("Server: %s\n\n", cfg.Server)

	for _, tc := range cfg.Tunnels {
		c := client.NewClient(cfg.Server, tc.ID, tc.Local, cfg.Token)
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
		fmt.Printf("- %s: https://%s/t/%s/ -> %s\n", tc.ID, cfg.Server, tc.ID, tc.Local)
	}

	<-interrupt
	fmt.Println("\nShutting down all tunnels...")
	for _, c := range clients {
		c.Stop()
	}
	wg.Wait()
}
