package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hacklab/internal/store"

	"github.com/spf13/cobra"
)

// aliasMap maps shorthand namespaces to their full git repo URLs.
// When a user runs "hacklab add <alias>/<path>", the alias prefix is
// expanded to the full repo URL with the path as a subdirectory.
//
// Examples:
//   examples/sqli-lab   →  https://github.com/HackLab-cli/lab-examples#examples/sqli-lab
//   examples/juice-shop →  https://github.com/HackLab-cli/lab-examples#examples/juice-shop
//   official/dvna       →  https://github.com/HackLab-cli/official-labs#official/dvna
//   official/dvwa       →  https://github.com/HackLab-cli/official-labs#official/dvwa
var aliasMap = map[string]string{
	"examples": "https://github.com/HackLab-cli/lab-examples",
	"official": "https://github.com/HackLab-cli/official-labs",
}

// resolveAlias checks if source matches a known alias prefix (e.g. "examples/sqli-lab").
// Returns the expanded repo URL and subdir if matched, empty strings otherwise.
func resolveAlias(source string) (repoURL, subdir string, matched bool) {
	// Must be a simple path-like string (no scheme, no leading ./ or /)
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "/") ||
		strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") ||
		strings.HasPrefix(source, "git@") {
		return "", "", false
	}

	// Split by "/" to get the namespace (first segment)
	parts := strings.SplitN(source, "/", 2)
	if len(parts) < 2 {
		return "", "", false
	}
	prefix := parts[0]

	repo, ok := aliasMap[prefix]
	if !ok {
		return "", "", false
	}

	return repo, source, true
}

var addCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "Add a lab from git repo, subdirectory, or local path",
	Long: `Add a lab to your collection.

Examples:
  # Full repo as a lab
  hacklab add https://github.com/user/my-hacklab

  # Specific subdirectory (repo#path)
  hacklab add https://github.com/HackLab-cli/lab-examples#labs/juice-shop

  # Official lab shorthand
  hacklab add official/dvwa
  hacklab add official/dvna

  # Example lab shorthand
  hacklab add examples/sqli-lab

  # Local folder
  hacklab add ./my-labs/sqli-lab
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

		if err := store.Ensure(); err != nil {
			return err
		}

		// Resolve alias shorthand (e.g. "examples/sqli-lab" → full git URL)
		if repoURL, subdir, ok := resolveAlias(source); ok {
			return addFromGit(repoURL, subdir)
		}

		// Parse #subdir syntax early, before deciding git vs local
		sourcePath := source
		var subdir string
		if idx := strings.Index(source, "#"); idx >= 0 {
			sourcePath = source[:idx]
			subdir = strings.TrimPrefix(source[idx+1:], "/")
		}

		isGit := strings.HasPrefix(sourcePath, "http://") ||
			strings.HasPrefix(sourcePath, "https://") ||
			strings.HasPrefix(sourcePath, "git@")

		if isGit {
			return addFromGit(sourcePath, subdir)
		}
		return addFromLocal(sourcePath, subdir)
	},
}

// addFromGit handles git URLs, with optional subdir
func addFromGit(repoURL, subdir string) error {
	// Determine lab name
	var name string
	if subdir != "" {
		parts := strings.Split(strings.TrimSuffix(subdir, "/"), "/")
		name = parts[len(parts)-1]
	} else {
		parts := strings.Split(strings.TrimSuffix(repoURL, "/"), "/")
		rawName := parts[len(parts)-1]
		name = strings.TrimSuffix(rawName, ".git")
	}

	labsDir, err := store.LabsDir()
	if err != nil {
		return err
	}
	destPath := filepath.Join(labsDir, name)

	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("lab '%s' already exists — remove it first", name)
	}

	if subdir == "" {
		fmt.Printf("  📦 cloning %s ...\n", repoURL)
		cloneCmd := exec.Command("git", "clone", "--depth", "1", repoURL, destPath)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr
		if err := cloneCmd.Run(); err != nil {
			return fmt.Errorf("git clone failed: %w", err)
		}
	} else {
		tmpDir, err := os.MkdirTemp("", "hacklab-clone-")
		if err != nil {
			return fmt.Errorf("creating temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		fmt.Printf("  📦 cloning %s ...\n", repoURL)
		cloneCmd := exec.Command("git", "clone", "--depth", "1", repoURL, tmpDir)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr
		if err := cloneCmd.Run(); err != nil {
			return fmt.Errorf("git clone failed: %w", err)
		}

		subPath := filepath.Join(tmpDir, subdir)
		if _, err := os.Stat(subPath); os.IsNotExist(err) {
			return fmt.Errorf("subdirectory '%s' not found in repo", subdir)
		}

		fmt.Printf("  📁 extracting lab from %s#%s ...\n", repoURL, subdir)
		if err := copyDir(subPath, destPath); err != nil {
			return fmt.Errorf("copying lab: %w", err)
		}
	}

	printSuccess(name, destPath)
	return nil
}

func addFromLocal(localPath, subdir string) error {
	src := localPath
	if subdir != "" {
		src = filepath.Join(localPath, subdir)
	}

	absPath, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	name := filepath.Base(absPath)

	labsDir, err := store.LabsDir()
	if err != nil {
		return err
	}
	destPath := filepath.Join(labsDir, name)

	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("lab '%s' already exists — remove it first", name)
	}

	if err := copyDir(absPath, destPath); err != nil {
		return fmt.Errorf("copying lab: %w", err)
	}

	printSuccess(name, destPath)
	return nil
}

func printSuccess(name, path string) {
	fmt.Println()
	fmt.Printf("  ⚡  hacklab: lab added\n")
	fmt.Printf("  🎯 name:     %s\n", name)
	fmt.Printf("  📁 location: %s\n", path)
	fmt.Println()
	fmt.Printf("  start it with: hacklab start %s\n", name)
	fmt.Println()
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
