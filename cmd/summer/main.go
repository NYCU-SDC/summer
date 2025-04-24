package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	appName        = "summer"
	appVersion     = "0.1.0"
	scriptsDir     = "./scripts/" // the directory where scripts are stored on the local machine
	defaultRepo    = "https://github.com/NYCU-SDC/summer/"
	defaultBranch  = "build/init-project"
	scriptRegistry = "registry.json"
)

var (
	repoURL    string
	repoBranch string
	rootCmd    = &cobra.Command{
		Use:   appName,
		Short: "A tool to download and manage useful scripts",
		Long: `ScriptGet allows you to download, manage, and use helpful scripts
in various languages. It makes non-Go scripts available as commands
on your system.`,
		Version: appVersion,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&repoURL, "repo", "r", defaultRepo, "URL of the script repository")
	rootCmd.PersistentFlags().StringVarP(&repoBranch, "branch", "b", defaultBranch, "Branch of the script repository")

	// Initialize commands
	rootCmd.AddCommand(initCommand())
	rootCmd.AddCommand(getScriptCommand())
}

func initCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return initFileStructure()
		},
	}
	return cmd
}

func getScriptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getscript [script-name]",
		Short: "Download a script",
		Long:  "Download a script from the repository and make it executable",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scriptName := args[0]
			return downloadScriptFromGit(repoURL, repoBranch, "/resource/scripts/"+scriptName, scriptsDir+scriptName)
		},
	}
	return cmd
}

func downloadScriptFromGit(repoURL, repoBranch, scriptPath, outputPath string) error {
	// Create a temporary directory for Git operations
	tempDir, err := os.MkdirTemp("", "scriptget-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("Failed to remove temp directory (%s): %v\n", tempDir, err)
		}
	}() // Clean up when done

	// Initialize Git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repo: %w", err)
	}

	// Add remote
	cmd = exec.Command("git", "remote", "add", "origin", repoURL)
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// Enable sparse checkout
	cmd = exec.Command("git", "config", "core.sparseCheckout", "true")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable sparse checkout: %w", err)
	}

	// Specify which files/folders to checkout
	sparseConfigPath := filepath.Join(tempDir, ".git", "info", "sparse-checkout")
	if err := os.WriteFile(sparseConfigPath, []byte(scriptPath), 0644); err != nil {
		return fmt.Errorf("failed to write sparse-checkout config: %w", err)
	}

	// Pull the repository (only the specified files/folders)
	cmd = exec.Command("git", "pull", "--depth=1", "origin", repoBranch) // Assuming main branch
	cmd.Dir = tempDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull from repository: %w: %s", err, string(out))
	}

	// Get the script from the checked out repo
	scriptFullPath := filepath.Join(tempDir, scriptPath)
	scriptContent, err := os.ReadFile(scriptFullPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	// Write the script to the output path
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write script to output: %w", err)
	}

	return nil
}

func downloadAllScriptFromGit(repoURL, repoBranch, scriptFolderPath, outputPath string) error {
	// Create a temporary directory for Git operations
	tempDir, err := os.MkdirTemp("", "scriptget-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("Failed to remove temp directory (%s): %v\n", tempDir, err)
		}
	}() // Clean up when done

	// Initialize Git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repo: %w", err)
	}

	// Add remote
	cmd = exec.Command("git", "remote", "add", "origin", repoURL)
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// Enable sparse checkout
	cmd = exec.Command("git", "config", "core.sparseCheckout", "true")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable sparse checkout: %w", err)
	}

	// Specify which files/folders to checkout
	sparseConfigPath := filepath.Join(tempDir, ".git", "info", "sparse-checkout")
	if err := os.WriteFile(sparseConfigPath, []byte(scriptFolderPath), 0644); err != nil {
		return fmt.Errorf("failed to write sparse-checkout config: %w", err)
	}

	// Pull the repository (only the specified files/folders)
	cmd = exec.Command("git", "pull", "--depth=1", "origin", repoBranch) // Assuming main branch
	cmd.Dir = tempDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull from repository: %w: %s", err, string(out))
	}

	// Get the script from the checked out repo
	scriptFullPath := filepath.Join(tempDir, scriptFolderPath)
	err = CopyDir(scriptFullPath, outputPath)
	if err != nil {
		return fmt.Errorf("failed to copy script folder: %w", err)
	}

	return nil
}

func downloadExampleFromGit(repoURL, repoBranch, examplePath, outputPath string) error {
	// Create a temporary directory for Git operations
	tempDir, err := os.MkdirTemp("", "scriptget-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("Failed to remove temp directory (%s): %v\n", tempDir, err)
		}
	}() // Clean up when done

	// Initialize Git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repo: %w", err)
	}

	// Add remote
	cmd = exec.Command("git", "remote", "add", "origin", repoURL)
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// Enable sparse checkout
	cmd = exec.Command("git", "config", "core.sparseCheckout", "true")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable sparse checkout: %w", err)
	}

	// Specify which files/folders to checkout
	sparseConfigPath := filepath.Join(tempDir, ".git", "info", "sparse-checkout")
	if err := os.WriteFile(sparseConfigPath, []byte(examplePath), 0644); err != nil {
		return fmt.Errorf("failed to write sparse-checkout config: %w", err)
	}

	// Pull the repository (only the specified files/folders)
	cmd = exec.Command("git", "pull", "--depth=1", "origin", repoBranch) // Assuming main branch
	cmd.Dir = tempDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull from repository: %w: %s", err, string(out))
	}

	// Get the script from the checked out repo
	exampleFullPath := filepath.Join(tempDir, examplePath)
	exampleContent, err := os.ReadFile(exampleFullPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	// Write the script to the output path
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, exampleContent, 0755); err != nil {
		return fmt.Errorf("failed to write script to output: %w", err)
	}
	return nil
}

func initFileStructure() error {
	// create main.go
	if err := os.MkdirAll(filepath.Dir("./cmd/main.go"), 0755); err != nil {
		return fmt.Errorf("failed to create cmd/main.go: %w", err)
	}

	// create "internal" folder
	if err := os.Mkdir(filepath.Dir("./internal/"), 0755); err != nil {
		return fmt.Errorf("failed to create ./internal/: %w", err)
	}

	// grab all scripts
	if err := downloadAllScriptFromGit(repoURL, repoBranch, "/resource/scripts/", "./scripts"); err != nil {
		return fmt.Errorf("failed to download scripts: %w", err)
	}

	// get example main.go
	if err := downloadExampleFromGit(repoURL, repoBranch, "/example/main.txt", "./cmd/main.go"); err != nil {
		return fmt.Errorf("failed to download example: %w", err)
	}

	return nil
}

// CopyDir clones all regular files from src → dst, making directories
// as needed. Everything is copied with default perms (dirs 0755, files 0644).
func CopyDir(src, dst string) error {
	// WalkDir is available since Go 1.16.
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err // abort the walk on any error
		}

		rel, _ := filepath.Rel(src, path) // path relative to src
		target := filepath.Join(dst, rel) // destination path

		if d.IsDir() {
			// Make the directory (ignore "already exists" errors)
			return os.MkdirAll(target, 0o755)
		}

		// Ignore anything that isn’t a plain file
		if !d.Type().IsRegular() {
			return nil
		}

		// Copy the file contents
		return copyFileContents(path, target)
	})
}

// copyFileContents streams bytes from src → dst (truncating/creating dst).
func copyFileContents(srcFile, dstFile string) (err error) {
	in, err := os.Open(srcFile)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dstFile) // perms default to 0644
	if err != nil {
		return
	}
	defer func() {
		// If the copy fails, make sure we don’t leave a 0-byte file behind
		if err != nil {
			_ = os.Remove(dstFile)
		}
		out.Close()
	}()

	_, err = io.Copy(out, in)
	return
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
