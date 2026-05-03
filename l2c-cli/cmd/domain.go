package cmd

import (
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

var domainCmd = &cobra.Command{
	Use:   "domain <custom-domain>",
	Short: "Update the custom domain for the tunneling ",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		customDomain := strings.TrimSpace(args[0])
		customDomain = strings.TrimPrefix(customDomain, "http://")
		customDomain = strings.TrimPrefix(customDomain, "https://")
		customDomain = strings.TrimRight(customDomain, "/")

		if !strings.Contains(customDomain, ".") {
			fmt.Println("Error: Invalid domain format. Must contain a dot (e.g., api.example.com).")
			return
		}

		// 1. Read existing config
		home, _ := os.UserHomeDir()
		configPath := filepath.Join(home, ".l2c", "config.json")

		data, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Printf("Error: config file not found at %s. Please run 'l2c setup' first.\n", configPath)
			return
		}

		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			fmt.Printf("Error parsing config: %v\n", err)
			return
		}

		if cfg.Token == "" {
			fmt.Println("Error: No auth token found in config. Please run 'l2c setup' first.")
			return
		}

		// 2. Check dependencies (wrangler)
		fmt.Print("Checking for wrangler... ")
		wranglerBin, err := exec.LookPath("wrangler")
		useNpx := false
		if err != nil {
			if _, nerr := exec.LookPath("npx"); nerr == nil {
				fmt.Println("OK (via npx)")
				useNpx = true
			} else {
				fmt.Println("\nError: 'wrangler' is not installed and 'npx' was not found.")
				return
			}
		} else {
			fmt.Println("OK")
		}

		runWrangler := func(cmdArgs ...string) *exec.Cmd {
			var cmdExec *exec.Cmd
			if useNpx {
				fullArgs := append([]string{"--yes", "wrangler"}, cmdArgs...)
				cmdExec = exec.Command("npx", fullArgs...)
			} else {
				cmdExec = exec.Command(wranglerBin, cmdArgs...)
			}
			cmdExec.Env = append(os.Environ(), "CI=true")
			return cmdExec
		}

		// 2.5 Check Login Status
		fmt.Print("Checking Cloudflare login status... ")
		whoamiCmd := runWrangler("whoami")
		if err := whoamiCmd.Run(); err != nil {
			fmt.Println("\nError: You are not logged into Cloudflare.")
			fmt.Println("Please run 'npx wrangler login' and then try again.")
			return
		}
		fmt.Println("OK")

		// 3. Extract Worker to Temp Dir
		fmt.Println("Preparing Worker deployment with new custom domain...")
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
			fileData, err := workerFS.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(targetPath, fileData, 0644)
		})
		if err != nil {
			fmt.Printf("Error extracting worker: %v\n", err)
			return
		}

		// 4. Inject Token and Custom Domain into wrangler.toml
		tomlPath := filepath.Join(tempDir, "wrangler.toml")
		tomlData, err := os.ReadFile(tomlPath)
		if err == nil {
			content := string(tomlData)
			content = strings.Replace(content, `AUTH_TOKEN = ""`, fmt.Sprintf(`AUTH_TOKEN = "%s"`, cfg.Token), 1)
			content = strings.Replace(content, `AUTH_TOKEN=""`, fmt.Sprintf(`AUTH_TOKEN="%s"`, cfg.Token), 1)

			// Append the custom domain routes block
			content += fmt.Sprintf("\n\n[[routes]]\npattern = \"%s\"\ncustom_domain = true\n", customDomain)

			os.WriteFile(tomlPath, []byte(content), 0644)
		}

		// 5. Deploy Worker
		fmt.Printf("Deploying Cloudflare Worker to %s...\n", customDomain)
		deployCmd := runWrangler("deploy")
		deployCmd.Dir = tempDir

		var deployOut strings.Builder
		deployCmd.Stdout = io.MultiWriter(os.Stdout, &deployOut)
		deployCmd.Stderr = os.Stderr

		if err := deployCmd.Run(); err != nil {
			fmt.Printf("\nError deploying worker: %v\n", err)
			fmt.Println("Make sure you have an active Cloudflare account, are logged in, and the domain is valid in your Cloudflare account.")
			return
		}

		// 6. Update config file
		cfg.CustomDomain = customDomain

		file, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(configPath, file, 0600); err != nil {
			fmt.Printf("Error writing updated config to %s: %v\n", configPath, err)
			return
		}

		fmt.Printf("\nSuccess! Custom domain updated to: %s\n", customDomain)
		fmt.Println("You can now run 'l2c run' to start your tunnel.")
	},
}

func init() {
	rootCmd.AddCommand(domainCmd)
}
