package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Tutorial Step Definition ─────────────────────────────────────────

type tutorialStep struct {
	Title        string
	Instructions string   // shown in the instructions area
	ExpectedCmd  string   // the command the user should type ("" = freeform, any key advances)
	Execute      bool     // actually run the command via os/exec
	MockOutput   []string // shown when Execute=false
}

var steps = []tutorialStep{
	{
		Title:        "Welcome to Hacklab!",
		Instructions: "Hacklab is your terminal hacking playground — spin up vulnerable labs,\ntrack objectives, and hack at your own pace.\n\nThis tutorial walks you through the 3 essential commands.\n\nPress Enter to begin.",
		ExpectedCmd:  "",
		Execute:      false,
	},
	{
		Title:        "Step 1 — List Your Labs",
		Instructions: "First, let's see what labs you have installed.\n\nType the command below and press Enter:",
		ExpectedCmd:  "hacklab list",
		Execute:      true,
	},
	{
		Title:        "Step 2 — Add an Example Lab",
		Instructions: "No labs yet? No problem! Add one from the official examples.\n\nAvailable example labs:\n  • juice-shop    — OWASP Juice Shop (beginner)\n  • sqli-lab       — SQL Injection Lab (beginner)\n  • xss-lab        — XSS Playground (beginner)\n  • jwt-lab        — JWT Auth Bypass (intermediate)\n\nInstall one now — type the command below:",
		ExpectedCmd:  "hacklab add examples/juice-shop",
		Execute:      true,
	},
	{
		Title:        "Step 3 — List Your Labs Again",
		Instructions: "Now you have a lab installed! List your labs again\nto see it appear:",
		ExpectedCmd:  "hacklab list",
		Execute:      true,
	},
	{
		Title:        "Step 4 — Start Hacking!",
		Instructions: "Once you've got a lab, fire it up! This launches Docker\ncontainers and opens an interactive challenge tracker\nwhere you mark objectives as you hack.\n\nProgress saves automatically to ~/.hacklab/progress.json.\n\nType the command below to see what happens:",
		ExpectedCmd:  "hacklab start juice-shop",
		Execute:      false,
		MockOutput: []string{
			"📦 Pulling image bkimminich/juice-shop:latest...",
			"🚀 Starting container...",
			"",
			"✅ Lab 'juice-shop' is running",
			"📡 Target: http://localhost:3000",
			"",
			"✅ Lab is ready",
			"",
			"⚡ OWASP Juice Shop",
			"📡 http://localhost:3000",
			"",
			"0/5  ░░░░░░░░░░░░░░░░░░░░   0%",
			"─────────────────────────────────────",
			"▸ ○  Bypass login with SQL injection   [injection]",
			"  ○  Steal admin JWT token             [auth]",
			"  ○  Access admin panel                [broken-auth]",
			"  ○  Find forgotten backup file        [sensitive-data]",
			"  ○  Exploit directory traversal       [injection]",
			"─────────────────────────────────────",
			"↑/↓ navigate  ·  space to toggle  ·  h hint  ·  q quit",
			"",
			"(Demo output — run it yourself after the tutorial!)",
		},
	},
	{
		Title: "You're Ready! ⚡",
		Instructions: "That's the basics! Here's your cheat sheet:\n\n" +
			"hacklab list           — see your installed labs\n" +
			"hacklab add <source>   — install a lab\n" +
			"hacklab start <name>   — launch a lab\n" +
			"hacklab status         — check what's running\n" +
			"hacklab stop <name>    — tear down a lab\n" +
			"hacklab remove <name>  — delete a lab\n\n" +
			"Labs are just folders with a lab.yml file.\n" +
			"Create your own and share them as git repos!\n\n" +
			"Happy hacking! 🔥",
		ExpectedCmd: "",
		Execute:     false,
	},
}

// ─── Terminal Output Entry ────────────────────────────────────────────

type termEntry struct {
	prompt  string // the prompt + command (e.g. "$ hacklab list")
	output  string // the command output (may be multi-line)
	isError bool
}

// ─── Model ────────────────────────────────────────────────────────────

type tutorialModel struct {
	step     int
	width    int
	height   int
	quitting bool

	// Terminal state
	input      string      // current input buffer
	cursorPos  int         // cursor position within input
	history    []termEntry // all past terminal entries
	termScroll int         // scroll offset in terminal history (0 = bottom)

	// Execution state
	phase   tutorialPhase
	execCmd *exec.Cmd
	spinIdx int

	// Binary path
	binaryPath string
}

type tutorialPhase int

const (
	phaseTyping     tutorialPhase = iota // user is typing a command
	phaseExecuting                       // command is running
	phaseShowResult                      // showing result, waiting for next step
)

// spinner frames
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type tickMsg struct{}
type execDoneMsg struct {
	output string
	err    error
}

func (m tutorialModel) Init() tea.Cmd {
	return findBinary()
}

func findBinary() tea.Cmd {
	return func() tea.Msg {
		// Find the hacklab binary — try os.Executable first, then PATH
		exe, err := os.Executable()
		if err == nil {
			return exe
		}
		p, err := exec.LookPath("hacklab")
		if err == nil {
			return p
		}
		return ""
	}
}

// ─── Update ───────────────────────────────────────────────────────────

func (m tutorialModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case string:
		// Binary path found
		m.binaryPath = msg
		return m, nil

	case tickMsg:
		if m.phase == phaseExecuting {
			m.spinIdx = (m.spinIdx + 1) % len(spinnerFrames)
			return m, tickCmd()
		}
		return m, nil

	case execDoneMsg:
		m.phase = phaseShowResult
		m.execCmd = nil

		output := msg.output
		if msg.err != nil {
			output = fmt.Sprintf("error: %v", msg.err)
		}

		m.history = append(m.history, termEntry{
			prompt:  "$ " + m.input,
			output:  output,
			isError: msg.err != nil,
		})

		m.input = ""
		m.cursorPos = 0
		m.termScroll = 0 // auto-scroll to bottom
		return m, nil

	case tea.KeyMsg:
		// Quit
		if msg.String() == "ctrl+c" {
			m.quitting = true
			m.killExec()
			return m, tea.Quit
		}

		switch m.phase {
		case phaseTyping:
			return m.handleTyping(msg)

		case phaseShowResult:
			return m.handleResult(msg)
		}
	}
	return m, nil
}

func (m *tutorialModel) handleTyping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.termScroll = 0 // reset scroll on submit
		return m.submitCommand()

	case "up":
		// Scroll terminal history up (show older entries)
		maxScroll := m.maxTermScroll()
		if m.termScroll < maxScroll {
			m.termScroll++
		}
		return m, nil

	case "down":
		// Scroll terminal history down (toward latest)
		if m.termScroll > 0 {
			m.termScroll--
		}
		return m, nil

	case "backspace":
		if m.cursorPos > 0 {
			m.input = m.input[:m.cursorPos-1] + m.input[m.cursorPos:]
			m.cursorPos--
		}
		return m, nil

	case "left":
		if m.cursorPos > 0 {
			m.cursorPos--
		}
		return m, nil

	case "right":
		if m.cursorPos < len(m.input) {
			m.cursorPos++
		}
		return m, nil

	case "home":
		m.cursorPos = 0
		return m, nil

	case "end":
		m.cursorPos = len(m.input)
		return m, nil

	default:
		// Regular character input
		if len(msg.String()) == 1 {
			ch := msg.String()[0]
			if ch >= 32 && ch < 127 {
				m.input = m.input[:m.cursorPos] + string(ch) + m.input[m.cursorPos:]
				m.cursorPos++
			}
		}
		return m, nil
	}
}

func (m *tutorialModel) handleResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.step < len(steps)-1 {
			m.step++
			m.input = ""
			m.cursorPos = 0
			m.phase = phaseTyping
			m.history = nil // clean terminal for next step
			m.termScroll = 0
		} else {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case "q":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *tutorialModel) submitCommand() (tea.Model, tea.Cmd) {
	step := steps[m.step]
	cmdStr := strings.TrimSpace(m.input)

	// If no expected command, empty input (Enter key) advances
	if step.ExpectedCmd == "" {
		if m.step < len(steps)-1 {
			m.step++
			m.input = ""
			m.cursorPos = 0
			m.history = nil // clean terminal for next step
			m.termScroll = 0
		} else {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	// Expected command but user pressed Enter with empty input — ignore
	if m.input == "" {
		return m, nil
	}

	// Check if command matches expected
	if cmdStr != step.ExpectedCmd {
		// Wrong command — show hint in terminal
		m.history = append(m.history, termEntry{
			prompt:  "$ " + m.input,
			output:  fmt.Sprintf("Not quite! Try typing exactly:\n  %s", step.ExpectedCmd),
			isError: true,
		})
		m.input = ""
		m.cursorPos = 0
		m.termScroll = 0
		return m, nil
	}

	// Correct command!
	if step.Execute {
		return m.executeReal(cmdStr)
	}

	// Simulate output
	m.history = append(m.history, termEntry{
		prompt: "$ " + m.input,
		output: strings.Join(step.MockOutput, "\n"),
	})
	m.input = ""
	m.cursorPos = 0
	m.termScroll = 0
	m.phase = phaseShowResult
	return m, nil
}

func (m *tutorialModel) executeReal(cmdStr string) (tea.Model, tea.Cmd) {
	m.phase = phaseExecuting
	m.spinIdx = 0

	// Parse the command: "hacklab list" -> args = ["list"]
	parts := strings.Fields(cmdStr)
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// Use the discovered binary path, or fall back to "hacklab"
	bin := m.binaryPath
	if bin == "" {
		bin = "hacklab"
	}

	cmd := exec.Command(bin, args...)
	cmd.Stdin = nil
	m.execCmd = cmd

	return m, tea.Batch(
		runCommand(cmd),
		tickCmd(),
	)
}

func runCommand(cmd *exec.Cmd) tea.Cmd {
	return func() tea.Msg {
		out, err := cmd.CombinedOutput()
		return execDoneMsg{
			output: strings.TrimRight(string(out), "\n"),
			err:    err,
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*120, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *tutorialModel) killExec() {
	if m.execCmd != nil && m.execCmd.Process != nil {
		m.execCmd.Process.Kill()
	}
}

// ─── View ─────────────────────────────────────────────────────────────

func (m tutorialModel) View() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	// Content column width — clamped to terminal, never overflow
	colW := 66
	if w-6 < colW {
		colW = w - 6
	}
	if colW < 40 {
		colW = w // very narrow terminal, use full width
	}

	// Horizontal padding
	horizPad := (w - colW) / 2
	if horizPad < 0 {
		horizPad = 0
	}
	padStr := strings.Repeat(" ", horizPad)

	// Build content lines
	contentLines := m.buildContent(colW)

	// Terminal sizing: fixed overhead = topBorder(1) + separator(1) + prompt(1) + bottomBorder(1) = 4
	const termOverhead = 4
	// Spacing above terminal
	const termTopGap = 1
	usedHeight := len(contentLines) + termOverhead + termTopGap
	// Minimum 3 visible history lines
	minHistory := 3
	var termHistoryLines int
	if h-usedHeight >= minHistory {
		termHistoryLines = h - usedHeight
	} else {
		// Not enough room — shrink to fit, but keep at least minHistory
		termHistoryLines = minHistory
		if termHistoryLines > h-termOverhead-termTopGap {
			termHistoryLines = h - termOverhead - termTopGap
		}
		if termHistoryLines < 1 {
			termHistoryLines = 1
		}
	}
	termLines := m.buildTerminal(colW, termHistoryLines)

	allLines := append(contentLines, "")
	allLines = append(allLines, termLines...)

	// Vertical centering
	total := len(allLines)
	padTop := 0
	if h > total {
		padTop = (h - total) / 2
	}

	var b strings.Builder
	for i := 0; i < padTop; i++ {
		b.WriteString("\n")
	}
	for _, line := range allLines {
		b.WriteString(padStr + line + "\n")
	}

	return b.String()
}

// maxTermScroll returns how far the user can scroll up (in flat output lines)
func (m tutorialModel) maxTermScroll() int {
	total := 0
	for _, entry := range m.history {
		total++                                        // prompt line
		total += strings.Count(entry.output, "\n") + 1 // output lines
	}
	// visible is dynamic; use a floor of 3 so we don't over-restrict
	visible := 3
	if total <= visible {
		return 0
	}
	return total - visible
}

// buildContent builds the instructions area as a list of lines
func (m tutorialModel) buildContent(colW int) []string {
	step := steps[m.step]
	total := len(steps)

	accent := lipgloss.Color(accentColor)
	cyan := lipgloss.Color(cyanColor)
	bright := lipgloss.Color("#e0e0f0")
	dim := lipgloss.Color(dimColor)
	green := lipgloss.Color(greenColor)

	center := func(s string) string {
		return lipgloss.NewStyle().Width(colW).Align(lipgloss.Center).Render(s)
	}

	var lines []string

	// Step dots
	lines = append(lines, center(m.buildStepDots(total, colW)))
	lines = append(lines, "")

	// Title
	titleStyle := lipgloss.NewStyle().Foreground(cyan).Bold(true)
	lines = append(lines, center(titleStyle.Render(step.Title)))
	lines = append(lines, "")

	// Instructions — wrap long lines
	instrStyle := lipgloss.NewStyle().Foreground(bright)
	for _, line := range strings.Split(step.Instructions, "\n") {
		lines = append(lines, center(instrStyle.Render(line)))
	}
	lines = append(lines, "")

	// Command box
	if step.ExpectedCmd != "" {
		cmdLabelStyle := lipgloss.NewStyle().Foreground(dim).Italic(true)
		lines = append(lines, center(cmdLabelStyle.Render("type this command in the terminal below:")))
		lines = append(lines, "")

		cmdBoxStyle := lipgloss.NewStyle().
			Foreground(green).
			Background(lipgloss.Color("#0d1117")).
			Bold(true).
			Padding(0, 2)
		lines = append(lines, center(cmdBoxStyle.Render(step.ExpectedCmd)))
		lines = append(lines, "")
	}

	// Status message
	if m.phase == phaseShowResult {
		statusStyle := lipgloss.NewStyle().Foreground(accent).Bold(true)
		lines = append(lines, center(statusStyle.Render("✓  Done! Press Enter to continue")))
	} else if m.phase == phaseExecuting {
		statusStyle := lipgloss.NewStyle().Foreground(accent).Bold(true)
		lines = append(lines, center(statusStyle.Render(spinnerFrames[m.spinIdx]+"  Running...")))
	} else if m.phase == phaseTyping && step.ExpectedCmd != "" {
		cmdLabelStyle := lipgloss.NewStyle().Foreground(dim).Italic(true)
		lines = append(lines, center(cmdLabelStyle.Render("type the command above and press Enter")))
	}

	return lines
}

func (m tutorialModel) buildStepDots(total, w int) string {
	var dots []string
	for i := 0; i < total; i++ {
		if i == m.step {
			dots = append(dots, lipgloss.NewStyle().
				Foreground(lipgloss.Color(accentColor)).
				Bold(true).
				Render("●"))
		} else if i < m.step {
			dots = append(dots, lipgloss.NewStyle().
				Foreground(lipgloss.Color(greenColor)).
				Render("●"))
		} else {
			dots = append(dots, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#2a2a4a")).
				Render("○"))
		}
	}
	return lipgloss.NewStyle().
		Width(w).
		Align(lipgloss.Center).
		Render(strings.Join(dots, "  "))
}

// buildTerminal builds the pronounced terminal area as a list of lines.
// historyLines is the number of visible history rows (excluding borders/separator/prompt).
func (m tutorialModel) buildTerminal(colW, historyLines int) []string {
	innerW := colW - 4 // space inside side borders
	if innerW < 20 {
		innerW = 20
	}

	dim := lipgloss.Color(dimColor)
	bright := lipgloss.Color("#c0c0d0")
	green := lipgloss.Color(greenColor)
	accent := lipgloss.Color(accentColor)

	// Top border with title
	topDash := colW - 13
	if topDash < 1 {
		topDash = 1
	}
	topBorder := lipgloss.NewStyle().
		Foreground(accent).
		Render("+- Terminal " + strings.Repeat("-", topDash) + "+")

	// Bottom border
	botDash := colW - 2
	if botDash < 1 {
		botDash = 1
	}
	botBorder := lipgloss.NewStyle().
		Foreground(accent).
		Render("+" + strings.Repeat("-", botDash) + "+")

	// Side border
	sideStyle := lipgloss.NewStyle().Foreground(accent)
	leftBar := sideStyle.Render("|")
	rightBar := sideStyle.Render("|")
	// Empty row inside border
	emptyFill := colW - 2
	if emptyFill < 1 {
		emptyFill = 1
	}
	emptyRow := leftBar + strings.Repeat(" ", emptyFill) + rightBar

	var lines []string
	lines = append(lines, topBorder)

	// Collect all output lines from history (flattened)
	var historyFlat []string
	for _, entry := range m.history {
		historyFlat = append(historyFlat, entry.prompt)
		for _, outLine := range strings.Split(entry.output, "\n") {
			historyFlat = append(historyFlat, "  "+outLine)
		}
	}

	// Apply scroll offset: skip `m.termScroll` lines from the end
	visibleCount := historyLines
	start := len(historyFlat) - visibleCount - m.termScroll
	if start < 0 {
		start = 0
	}
	end := start + visibleCount
	if end > len(historyFlat) {
		end = len(historyFlat)
	}
	visibleHistory := historyFlat[start:end]

	// Pad to fill visible area
	for len(visibleHistory) < visibleCount {
		visibleHistory = append([]string{""}, visibleHistory...)
	}

	// Render each history line inside border
	for _, raw := range visibleHistory {
		if raw == "" {
			lines = append(lines, emptyRow)
			continue
		}
		styled := lipgloss.NewStyle().Foreground(bright).Render(raw)
		lines = append(lines, sideLine(styled, innerW, leftBar, rightBar))
	}

	// Scroll indicator if there's hidden history
	if m.termScroll > 0 {
		indicator := lipgloss.NewStyle().
			Foreground(dim).
			Italic(true).
			Render(fmt.Sprintf("↑ %d more lines ↑", m.termScroll))
		lines = append(lines, sideLine(indicator, innerW, leftBar, rightBar))
	}

	// Separator
	sepFill := colW - 2
	if sepFill < 1 {
		sepFill = 1
	}
	sep := leftBar + lipgloss.NewStyle().Foreground(dim).Render(strings.Repeat("-", sepFill)) + rightBar
	lines = append(lines, sep)

	// Prompt line with cursor, inside border
	promptStyled := buildPromptLine(m.input, m.cursorPos, m.phase == phaseExecuting, innerW, green, bright, dim)
	lines = append(lines, sideLine(promptStyled, innerW, leftBar, rightBar))

	lines = append(lines, botBorder)

	return lines
}

func buildPromptLine(input string, cursorPos int, isExec bool, w int, green, bright, dim lipgloss.Color) string {
	// Clamp input length so prompt + input + cursor fits within w
	overhead := 3 // "$ " + cursor char
	maxInput := w - overhead
	if maxInput < 0 {
		maxInput = 0
	}
	if len(input) > maxInput {
		input = input[:maxInput]
	}
	if cursorPos > len(input) {
		cursorPos = len(input)
	}

	prompt := lipgloss.NewStyle().Foreground(green).Bold(true).Render("$ ")

	var before, at, after string
	before = input[:cursorPos]
	if cursorPos < len(input) {
		at = string(input[cursorPos])
		after = input[cursorPos+1:]
	} else {
		at = " " // visible cursor at end
	}

	cmdStyle := lipgloss.NewStyle().Foreground(bright)
	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color(bright)).
		Bold(true)

	if isExec {
		cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color(accentColor)).
			Bold(true)
	}

	return prompt + cmdStyle.Render(before) + cursorStyle.Render(at) + cmdStyle.Render(after)
}

// sideLine wraps content between left and right borders, padding or
// truncating so the visual width never exceeds innerW.
func sideLine(content string, innerW int, left, right string) string {
	// Truncate if content is wider than innerW (ANSI-aware)
	constrained := lipgloss.NewStyle().MaxWidth(innerW).Render(content)
	vis := lipgloss.Width(constrained)
	pad := innerW - vis
	if pad < 0 {
		pad = 0
	}
	return left + " " + constrained + strings.Repeat(" ", pad) + " " + right
}

// ─── Public API ───────────────────────────────────────────────────────

func RunTutorial() error {
	prog := tea.NewProgram(
		tutorialModel{
			step:  0,
			phase: phaseTyping,
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := prog.Run()
	return err
}
