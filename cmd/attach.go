package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"hacklab/internal/docker"
	"hacklab/internal/lab"
	"hacklab/internal/progress"
	"hacklab/internal/store"
	"hacklab/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// attachable represents a running lab that can be attached to
type attachable struct {
	name      string
	lab       *lab.Lab
	targetURL string
}

var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach to a running lab",
	Long:  `Find running lab containers and resume an interactive session with existing progress.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find running labs
		running, err := docker.ListRunning()
		if err != nil {
			return err
		}

		if len(running) == 0 {
			fmt.Println("\n  no labs running — start one with 'hacklab start <name>'")
			return nil
		}

		// Deduplicate (compose labs produce multiple containers with the same label)
		seen := make(map[string]bool)
		var unique []string
		for _, name := range running {
			if !seen[name] {
				seen[name] = true
				unique = append(unique, name)
			}
		}
		running = unique

		// Build list of running labs with their manifests
		var attachables []attachable
		for _, name := range running {
			labPath, err := store.LabPath(name)
			if err != nil {
				continue
			}
			if _, err := os.Stat(labPath); os.IsNotExist(err) {
				// Lab container is running but lab files are missing
				fmt.Printf("  ⚠️  container 'hacklab-%s' is running but lab files not found — skipping\n", name)
				continue
			}

			l, err := lab.LoadLab(labPath)
			if err != nil {
				fmt.Printf("  ⚠️  failed to load lab '%s': %v — skipping\n", name, err)
				continue
			}

			targetURL := getContainerURL(name, l)

			attachables = append(attachables, attachable{
				name:      name,
				lab:       l,
				targetURL: targetURL,
			})
		}

		if len(attachables) == 0 {
			fmt.Println("\n  no attachable labs found")
			return nil
		}

		// If only one running lab, attach directly
		var chosen *attachable
		if len(attachables) == 1 {
			chosen = &attachables[0]
		} else {
			// Let user pick from multiple running labs
			chosen, err = selectRunningLab(attachables)
			if err != nil {
				return err
			}
			if chosen == nil {
				fmt.Println("\n  no lab selected")
				return nil
			}
		}

		// Load saved progress
		p, err := progress.Load()
		if err != nil {
			return err
		}

		fmt.Printf("\n  📎 Attaching to '%s'...\n", chosen.lab.Manifest.Name)
		if chosen.targetURL != "" {
			fmt.Printf("  📡 Target: %s\n", chosen.targetURL)
		}
		fmt.Println()

		// Launch TUI with existing progress
		return tui.AttachLab(chosen.lab, p, chosen.targetURL)
	},
}

// getContainerURL inspects running containers and builds the target URL.
// For compose labs it finds the web service container via the project name.
func getContainerURL(labName string, l *lab.Lab) string {
	// For single-container labs, inspect the named container
	if l.Manifest.Image != "" && l.Manifest.Port > 0 {
		containerName := fmt.Sprintf("hacklab-%s", labName)
		return inspectPort(containerName, l.Manifest.Port)
	}

	// For compose labs, find the web service container
	if l.Manifest.ComposeFile != "" {
		projectName := fmt.Sprintf("hacklab-%s", labName)
		// List containers in this compose project
		cmd := exec.Command("docker", "ps",
			"--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName),
			"--format", "{{.Names}}")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, name := range lines {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				// Try to find the web/app service (skip db, redis, etc.)
				if url := inspectPort(name, l.Manifest.Port); url != "" {
					return url
				}
			}
		}
		// Fallback: use wait_for URL if available
		if l.Manifest.WaitFor != "" {
			return l.Manifest.WaitFor
		}
	}

	return ""
}

// inspectPort inspects a single container's port binding and returns the URL.
// If port is 0, it tries common web ports and returns the first match.
func inspectPort(containerName string, port int) string {
	cmd := exec.Command("docker", "inspect", "-f",
		"{{json .NetworkSettings.Ports}}", containerName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	var ports map[string][]map[string]string
	if err := json.Unmarshal(output, &ports); err != nil {
		return ""
	}

	// Try the specified port first
	if port > 0 {
		if url := tryPortBinding(ports, port); url != "" {
			return url
		}
	}

	// Fallback: scan common web ports
	if url := tryPortBinding(ports, 80); url != "" {
		return url
	}
	if url := tryPortBinding(ports, 3000); url != "" {
		return url
	}
	if url := tryPortBinding(ports, 8080); url != "" {
		return url
	}
	if url := tryPortBinding(ports, 443); url != "" {
		return url
	}

	return ""
}

func tryPortBinding(ports map[string][]map[string]string, port int) string {
	containerPort := fmt.Sprintf("%d/tcp", port)
	if bindings, ok := ports[containerPort]; ok && len(bindings) > 0 {
		hostPort := bindings[0]["HostPort"]
		if hostPort != "" {
			return fmt.Sprintf("http://localhost:%s", hostPort)
		}
	}
	return ""
}

// selectRunningLab presents an interactive picker for multiple running labs
func selectRunningLab(labs []attachable) (*attachable, error) {
	m := &attachSelectModel{
		labs:   labs,
		cursor: 0,
		done:   false,
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	final := result.(*attachSelectModel)
	if final.chosen < 0 || final.chosen >= len(labs) {
		return nil, nil
	}
	return &labs[final.chosen], nil
}

type attachSelectModel struct {
	labs   []attachable
	cursor int
	chosen int
	done   bool
}

func (m attachSelectModel) Init() tea.Cmd {
	return nil
}

func (m attachSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.chosen = -1
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.labs)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.chosen = m.cursor
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m attachSelectModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  ⚡  select a running lab\n\n")

	for i, a := range m.labs {
		arrow := "  "
		if i == m.cursor {
			arrow = "▸ "
		}

		name := a.lab.Manifest.Name
		if i == m.cursor {
			name = fmt.Sprintf("\033[1m%s\033[0m", name)
		}

		desc := ""
		if a.lab.Manifest.Description != "" {
			desc = " — " + a.lab.Manifest.Description
		}

		typeLabel := ""
		if a.lab.Manifest.ComposeFile != "" {
			typeLabel = " [compose]"
		}

		b.WriteString(fmt.Sprintf("  %s%s%s%s\n", arrow, name, typeLabel, desc))
	}

	b.WriteString("\n")
	b.WriteString("  ↑/↓ select  ·  enter attach  ·  q cancel\n")
	b.WriteString("\n")

	return b.String()
}
