package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TutorialStep represents a single step in the tutorial
type TutorialStep struct {
	Title       string
	Command     string   // the command being taught
	CommandDesc string   // what the command does
	Explanation string   // longer explanation
	DemoLines   []string // mock output to show
}

// Available example labs for the tutorial (matching the lab-examples repo)
var exampleLabs = []struct {
	Slug        string
	Name        string
	Description string
	Difficulty  string
}{
	{"juice-shop", "OWASP Juice Shop", "Modern web app with 5 vulnerabilities to exploit", "beginner"},
	{"sqli-lab", "SQL Injection Lab", "PHP + MySQL lab to practice SQLi techniques", "beginner"},
	{"xss-lab", "XSS Playground", "Reflected and stored XSS challenges", "beginner"},
	{"jwt-lab", "JWT Auth Bypass", "Exploit JWT misconfigurations and forge tokens", "intermediate"},
}

var tutorialSteps = []TutorialStep{
	{
		Title:       "Welcome to Hacklab!",
		Command:     "",
		CommandDesc: "",
		Explanation: "Hacklab is your terminal hacking playground. Spin up vulnerable lab environments with one command, track your objectives, and hack at your own pace — all from the terminal.\n\nThis quick tutorial will show you the 3 essential commands to get started.",
		DemoLines:   nil,
	},
	{
		Title:       "List Your Labs",
		Command:     "hacklab list",
		CommandDesc: "Show all installed labs",
		Explanation: "Once you've added labs, running 'hacklab list' shows what's available — their names, descriptions, difficulty, and how many objectives they contain. Right now you have no labs installed, but that changes next!",
		DemoLines: []string{
			"",
			"  no labs found — add one with 'hacklab add <source>'",
			"",
		},
	},
	{
		Title:       "Add Example Labs",
		Command:     "hacklab add examples/<lab-name>",
		CommandDesc: "Install a lab from the official examples",
		Explanation: "Hacklab ships with example labs you can add using a shorthand alias. Just pick one from the list below!\n\nAvailable example labs:",
		DemoLines: []string{
			"",
			"  📦 cloning https://github.com/HackLab-cli/lab-examples ...",
			"  📁 extracting lab from examples/sqli-lab ...",
			"",
			"  ⚡  hacklab: lab added",
			"  🎯 name:     sqli-lab",
			"  📁 location: ~/.hacklab/labs/sqli-lab",
			"",
			"  start it with: hacklab start sqli-lab",
			"",
		},
	},
	{
		Title:       "Start Hacking!",
		Command:     "hacklab start <lab-name>",
		CommandDesc: "Launch a lab with its interactive challenge tracker",
		Explanation: "After adding a lab, start it to spin up the Docker containers and launch the interactive TUI. You'll get a checklist of objectives to complete — mark them off as you hack. Progress saves automatically!",
		DemoLines: []string{
			"",
			"  📦 Pulling image vulnerables/web-dvwa:latest...",
			"  🚀 Starting container...",
			"",
			"  ✅ Lab 'sqli-lab' is running",
			"  📡 Target: http://localhost:8080",
			"",
			"  ✅ Lab is ready",
			"",
			"  ⚡ SQL Injection Lab",
			"  📡 http://localhost:8080",
			"",
			"  0/5  ░░░░░░░░░░░░░░░░░░░░  0%",
			"  ─────────────────────────────────────",
			"  ▸ ○  Bypass login authentication   [sqli]",
			"    ○  Dump user credentials         [sqli]",
			"    ○  Extract password hashes       [sqli]",
			"  ─────────────────────────────────────",
			"",
		},
	},
	{
		Title:       "You're Ready!",
		Command:     "",
		CommandDesc: "",
		Explanation: "That's it! Here's your cheat sheet:\n\n  • hacklab list          — see your labs\n  • hacklab add <source>  — install a lab\n  • hacklab start <name>  — launch a lab\n  • hacklab status        — check running labs\n  • hacklab stop <name>   — tear down a lab\n  • hacklab remove <name> — delete a lab\n\nLabs are just folders with a lab.yml file. Create your own and share them!\n\nHappy hacking! ⚡",
		DemoLines:   nil,
	},
}

// Tutorial model
type tutorialModel struct {
	step    int
	width   int
	height  int
	quitting bool
}

func (m tutorialModel) Init() tea.Cmd {
	return nil
}

func (m tutorialModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "right", "n", "enter", " ":
			if m.step < len(tutorialSteps)-1 {
				m.step++
			} else {
				m.quitting = true
				return m, tea.Quit
			}

		case "left", "p":
			if m.step > 0 {
				m.step--
			}
		}
	}
	return m, nil
}

func (m tutorialModel) View() string {
	w := m.width
	if w <= 0 {
		w = 80
	}

	step := tutorialSteps[m.step]
	totalSteps := len(tutorialSteps)

	var b strings.Builder

	// === TOP BAR: Step indicator ===
	stepBar := m.buildStepBar(totalSteps, w)
	b.WriteString(stepBar + "\n")

	// === CONTENT ===
	content := m.buildStepContent(step, w)

	// Vertical centering for content (approximate)
	totalContent := strings.Count(content, "\n") + 1
	footer := m.buildFooter(totalSteps, w)
	footerLines := strings.Count(footer, "\n") + 1
	availableHeight := m.height - 3 - footerLines // stepBar(1) + 2 spacing
	if availableHeight < 8 {
		availableHeight = 8
	}

	padTop := 0
	if availableHeight > totalContent {
		padTop = (availableHeight - totalContent) / 2
	}

	b.WriteString(strings.Repeat("\n", padTop))
	b.WriteString(content)

	// Push footer to bottom
	remaining := m.height - padTop - totalContent - footerLines - 1
	if remaining > 0 {
		b.WriteString(strings.Repeat("\n", remaining))
	}
	b.WriteString(footer)

	return b.String()
}

func (m tutorialModel) buildStepBar(totalSteps, w int) string {
	var segments []string
	for i := 0; i < totalSteps; i++ {
		num := fmt.Sprintf("%d", i+1)
		if i == m.step {
			segments = append(segments,
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("#000000")).
					Background(lipgloss.Color(accentColor)).
					Bold(true).
					Padding(0, 1).
					Render(" "+num+" "),
			)
		} else if i < m.step {
			segments = append(segments,
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("#000000")).
					Background(lipgloss.Color(greenColor)).
					Bold(true).
					Padding(0, 1).
					Render(" "+num+" "),
			)
		} else {
			segments = append(segments,
				lipgloss.NewStyle().
					Foreground(lipgloss.Color(dimColor)).
					Background(lipgloss.Color("#1a1a2e")).
					Padding(0, 1).
					Render(" "+num+" "),
			)
		}
	}

	joined := strings.Join(segments, "─")
	padded := lipgloss.NewStyle().
		Width(w).
		Align(lipgloss.Center).
		Render(joined)

	return padded
}

func (m tutorialModel) buildStepContent(step TutorialStep, w int) string {
	innerW := w - 8
	if innerW < 40 {
		innerW = 40
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(cyanColor)).
		Bold(true).
		Width(innerW).
		Align(lipgloss.Center)

	// Command display
	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff88")).
		Background(lipgloss.Color("#0d1117")).
		Bold(true).
		Padding(1, 2).
		Width(innerW).
		Align(lipgloss.Center)

	cmdDescStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(dimColor)).
		Width(innerW).
		Align(lipgloss.Center)

	explanationStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d0d0e0")).
		Width(innerW).
		Align(lipgloss.Center)

	boxBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 2).
		Width(innerW + 4)

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render(step.Title) + "\n")
	b.WriteString("\n")

	// Command section
	if step.Command != "" {
		// Render command first, then command desc below
		b.WriteString(cmdStyle.Render(step.Command) + "\n")
		b.WriteString(cmdDescStyle.Render(step.CommandDesc) + "\n")
		b.WriteString("\n")
	}

	// Explanation
	if step.Explanation != "" {
		for _, line := range strings.Split(step.Explanation, "\n") {
			b.WriteString(explanationStyle.Render(line) + "\n")
		}
		b.WriteString("\n")
	}

	// Example labs grid (for step 3)
	if step.Title == "Add Example Labs" {
		b.WriteString(m.buildExampleGrid(innerW) + "\n")
		b.WriteString("\n")

		// Show commands to try
		tryStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(yellowColor)).
			Bold(true).
			Width(innerW).
			Align(lipgloss.Center)

		b.WriteString(tryStyle.Render("Try:") + "\n")
		tryCmdStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff88")).
			Background(lipgloss.Color("#0d1117")).
			Padding(0, 1).
			Width(innerW).
			Align(lipgloss.Center)
		b.WriteString(tryCmdStyle.Render("hacklab add examples/juice-shop") + "\n")
		b.WriteString(tryCmdStyle.Render("hacklab add examples/sqli-lab") + "\n")
		b.WriteString("\n")
	}

	// Demo output
	if len(step.DemoLines) > 0 {
		demoHeader := lipgloss.NewStyle().
			Foreground(lipgloss.Color(dimColor)).
			Italic(true).
			Render("── demo ──")
		demoContent := strings.Join(step.DemoLines, "\n")

		boxContent := demoHeader + "\n" + demoContent
		b.WriteString(boxBorderStyle.Render(boxContent) + "\n")
	}

	return b.String()
}

func (m tutorialModel) buildExampleGrid(w int) string {
	var b strings.Builder

	cardBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(cyanColor)).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a0a0b8"))

	tagStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(accentColor))

	for _, lab := range exampleLabs {
		line := fmt.Sprintf("%s  %s  %s",
			headerStyle.Render(lab.Slug),
			descStyle.Render(lab.Description),
			tagStyle.Render("["+lab.Difficulty+"]"),
		)
		// Truncate/pad for alignment
		b.WriteString(cardBorder.Render(line) + "\n")
	}

	return b.String()
}

func (m tutorialModel) buildFooter(totalSteps, w int) string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(dimColor)).
		Width(w).
		Align(lipgloss.Center)

	nav := "←/→ or n/p navigate  ·  enter/space advance  ·  q quit"
	if m.step == totalSteps-1 {
		nav = "enter or space to finish  ·  q quit"
	}

	return footerStyle.Render(nav)
}

// RunTutorial starts the tutorial TUI
func RunTutorial() error {
	prog := tea.NewProgram(
		tutorialModel{step: 0},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := prog.Run()
	return err
}
