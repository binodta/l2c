package cmd

import (
	"bufio"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed all:worker_src
var workerFS embed.FS

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup for Cloudflare Worker and CLI",
	Run: func(cmd *cobra.Command, args []string) {
		// Open /dev/tty to ensure we read from the keyboard even if stdin is a pipe
		tty, err := os.Open("/dev/tty")
		if err != nil {
			// Fallback to stdin if tty is not available
			tty = os.Stdin
		}
		defer tty.Close()
		reader := bufio.NewReader(tty)

		fmt.Println("l2c - Cloudflare Tunnel Setup")
		fmt.Println("-------------------------------")

		// 1. Check dependencies
		fmt.Print("Checking for wrangler... ")
		wranglerBin, err := exec.LookPath("wrangler")
		useNpx := false
		if err != nil {
			// Fallback to npx
			if _, nerr := exec.LookPath("npx"); nerr == nil {
				fmt.Println("OK (via npx)")
				useNpx = true
				wranglerBin = "npx"
			} else {
				fmt.Println("\nError: 'wrangler' is not installed and 'npx' was not found.")
				fmt.Println("Please install it with 'npm install -g wrangler'.")
				return
			}
		} else {
			fmt.Println("OK")
		}

		runWrangler := func(args ...string) *exec.Cmd {
			var cmd *exec.Cmd
			if useNpx {
				fullArgs := append([]string{"--yes", "wrangler"}, args...)
				cmd = exec.Command("npx", fullArgs...)
			} else {
				cmd = exec.Command(wranglerBin, args...)
			}
			// Add CI=true to avoid some interactive prompts
			cmd.Env = append(os.Environ(), "CI=true")
			return cmd
		}

		// 2. Check Login Status
		fmt.Print("Checking Cloudflare login status... ")
		whoamiCmd := runWrangler("whoami")
		if err := whoamiCmd.Run(); err != nil {
			fmt.Println("\nError: You are not logged into Cloudflare.")
			fmt.Println("Please run 'npx wrangler login' in another terminal and then try again.")
			return
		}
		fmt.Println("OK")

		// 3. Auth Token — auto-generated UUID v4, no user prompt needed.
		token, err := generateUUID()
		if err != nil {
			fmt.Printf("Error generating token: %v\n", err)
			return
		}
		fmt.Printf("Generated auth token: %s\n", token)

		// 4. Extract Worker to Temp Dir
		fmt.Println("\nPreparing Worker deployment...")
		if useNpx {
			fmt.Println("(Note: This may take a minute if npx needs to download wrangler)")
		}
		tempDir, err := os.MkdirTemp("", "l2c-worker-*")
		if err != nil {
			fmt.Printf("Error creating temp dir: %v\n", err)
			return
		}
		defer os.RemoveAll(tempDir)

		err = fs.WalkDir(workerFS, "worker_src", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel("worker_src", path)
			if relPath == "." {
				return nil
			}
			targetPath := filepath.Join(tempDir, relPath)
			if d.IsDir() {
				return os.MkdirAll(targetPath, 0755)
			}
			data, err := workerFS.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(targetPath, data, 0644)
		})
		if err != nil {
			fmt.Printf("Error extracting worker: %v\n", err)
			return
		}

		// 4.5 Prompt for Custom Domain
		fmt.Print("\nDo you want to configure a custom domain? (y/N): ")
		customDomainResp, _ := reader.ReadString('\n')
		customDomainResp = strings.TrimSpace(strings.ToLower(customDomainResp))
		
		customDomain := ""
		if customDomainResp == "y" || customDomainResp == "yes" {
			for {
				fmt.Print("Enter your custom domain (e.g., api.example.com): ")
				customDomain, _ = reader.ReadString('\n')
				customDomain = strings.TrimSpace(customDomain)
				
				customDomain = strings.TrimPrefix(customDomain, "http://")
				customDomain = strings.TrimPrefix(customDomain, "https://")
				customDomain = strings.TrimRight(customDomain, "/")
				
				if customDomain == "" {
					fmt.Println("Domain cannot be empty.")
					continue
				}
				if !strings.Contains(customDomain, ".") {
					fmt.Println("Invalid domain format. Must contain a dot (e.g., api.example.com).")
					continue
				}
				break
			}
		}

		// 4.6 Inject Token and Custom Domain into wrangler.toml
		tomlPath := filepath.Join(tempDir, "wrangler.toml")
		tomlData, err := os.ReadFile(tomlPath)
		if err == nil {
			content := string(tomlData)
			// Handle both AUTH_TOKEN = "" and AUTH_TOKEN=""
			content = strings.Replace(content, `AUTH_TOKEN = ""`, fmt.Sprintf(`AUTH_TOKEN = "%s"`, token), 1)
			content = strings.Replace(content, `AUTH_TOKEN=""`, fmt.Sprintf(`AUTH_TOKEN="%s"`, token), 1)
			
			if customDomain != "" {
				content += fmt.Sprintf("\n\n[[routes]]\npattern = \"%s\"\ncustom_domain = true\n", customDomain)
			}
			
			os.WriteFile(tomlPath, []byte(content), 0644)
		}

		// 5. Deploy Worker
		fmt.Println("Deploying Cloudflare Worker...")
		deployCmd := runWrangler("deploy")
		deployCmd.Dir = tempDir
		
		// Capture output to extract host
		var deployOut strings.Builder
		deployCmd.Stdout = io.MultiWriter(os.Stdout, &deployOut)
		deployCmd.Stderr = os.Stderr
		
		if err := deployCmd.Run(); err != nil {
			fmt.Printf("\nError deploying worker: %v\n", err)
			fmt.Println("Make sure you have an active Cloudflare account and are logged in.")
			return
		}

		// 6. Detect Host
		workerURL := ""
		lines := strings.Split(deployOut.String(), "\n")
		for _, line := range lines {
			if strings.Contains(line, "workers.dev") {
				parts := strings.Fields(line)
				for _, p := range parts {
					// Clean up any ANSI escape codes or extra characters
					p = strings.TrimSpace(p)
					if strings.HasPrefix(p, "https://") && strings.Contains(p, "workers.dev") {
						workerURL = strings.TrimPrefix(p, "https://")
						break
					}
				}
			}
		}

		host := customDomain
		if host == "" {
			host = workerURL
		}

		if host == "" {
			fmt.Printf("\nCould not automatically detect host.\nEnter your Cloudflare Worker host: ")
			host, _ = reader.ReadString('\n')
			host = strings.TrimSpace(host)
			
			if workerURL == "" && strings.Contains(host, "workers.dev") {
			    workerURL = host
			}
		}

		if host == "" {
			fmt.Println("Error: Host is required.")
			return
		}
		fmt.Printf("Using Worker host: %s\n", host)

		// 7. Create config file in Home Dir
		cfg := Config{
			WorkerURL:    workerURL,
			CustomDomain: customDomain,
			Token:        token,
			Tunnels: []TunnelConfig{
				{ID: "app-one", Local: "http://localhost:8000"},
			},
		}

		home, _ := os.UserHomeDir()
		configDir := filepath.Join(home, ".l2c")
		configPath := filepath.Join(configDir, "config.json")

		// Ensure directory exists
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			return
		}

		file, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(configPath, file, 0600); err != nil {
			fmt.Printf("Error writing config to %s: %v\n", configPath, err)
			return
		}

		fmt.Printf("\nSetup complete! Config saved to %s\n", configPath)
		fmt.Println("Run 'l2c run' to start the tunnel.")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

// generateUUID creates a random UUID v4 string.
func generateUUID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// UUID v4 format
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
