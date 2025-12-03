// Package tui provides the Bubble Tea-based terminal UI for Celeste CLI.
// This file contains the main application model and layout logic.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppModel is the root model for the Celeste TUI application.
type AppModel struct {
	// Sub-components
	header   HeaderModel
	chat     ChatModel
	input    InputModel
	skills   SkillsModel
	status   StatusModel

	// Application state
	width     int
	height    int
	ready     bool
	nsfwMode  bool
	streaming bool

	// LLM client (injected)
	llmClient LLMClient
}

// LLMClient interface for sending messages to the LLM.
type LLMClient interface {
	SendMessage(messages []ChatMessage, tools []SkillDefinition) tea.Cmd
	GetSkills() []SkillDefinition
}

// SkillDefinition represents a skill/function that can be called.
type SkillDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// NewApp creates a new TUI application model.
func NewApp(llmClient LLMClient) AppModel {
	skills := []SkillDefinition{}
	if llmClient != nil {
		skills = llmClient.GetSkills()
	}

	return AppModel{
		header:    NewHeaderModel(),
		chat:      NewChatModel(),
		input:     NewInputModel(),
		skills:    NewSkillsModel(skills),
		status:    NewStatusModel(),
		llmClient: llmClient,
	}
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.input.Init(),
		tea.EnterAltScreen,
	)
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "pgup", "pgdown", "shift+up", "shift+down":
			// Scrolling keys go to chat
			var cmd tea.Cmd
			m.chat, cmd = m.chat.Update(msg)
			cmds = append(cmds, cmd)
		default:
			// Other keys go to input
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Calculate component heights
		headerHeight := 1
		inputHeight := 3
		skillsHeight := 5
		statusHeight := 1
		chatHeight := m.height - headerHeight - inputHeight - skillsHeight - statusHeight

		// Update component sizes
		m.header = m.header.SetWidth(m.width)
		m.chat = m.chat.SetSize(m.width, chatHeight)
		m.input = m.input.SetWidth(m.width)
		m.skills = m.skills.SetSize(m.width, skillsHeight)
		m.status = m.status.SetWidth(m.width)

	case SendMessageMsg:
		// Handle special commands
		content := strings.TrimSpace(msg.Content)
		lowerContent := strings.ToLower(content)

		switch lowerContent {
		case "exit", "quit", "q", ":q", ":quit", ":exit":
			return m, tea.Quit
		case "clear":
			m.chat = m.chat.Clear()
			m.status = m.status.SetText("Chat cleared")
			return m, nil
		case "help":
			m.chat = m.chat.AddSystemMessage(helpText())
			return m, nil
		}

		// Add user message to chat
		m.chat = m.chat.AddUserMessage(content)
		m.streaming = true
		m.status = m.status.SetStreaming(true)

		// Send to LLM
		if m.llmClient != nil {
			cmds = append(cmds, m.llmClient.SendMessage(m.chat.GetMessages(), m.skills.GetDefinitions()))
		}

	case StreamChunkMsg:
		m.chat = m.chat.AppendToLastAssistant(msg.Chunk.Content)
		if msg.Chunk.IsFirst {
			m.chat = m.chat.AddAssistantMessage("")
		}
		cmds = append(cmds, nil) // Keep processing

	case StreamDoneMsg:
		m.streaming = false
		m.status = m.status.SetStreaming(false)
		m.status = m.status.SetText(fmt.Sprintf("Done (%s)", msg.FinishReason))

	case StreamErrorMsg:
		m.streaming = false
		m.status = m.status.SetStreaming(false)
		m.status = m.status.SetText(fmt.Sprintf("Error: %v", msg.Err))
		m.chat = m.chat.AddSystemMessage(fmt.Sprintf("Error: %v", msg.Err))

	case SkillCallMsg:
		m.skills = m.skills.SetExecuting(msg.Call.Name)
		m.chat = m.chat.AddFunctionCall(msg.Call)

	case SkillResultMsg:
		if msg.Err != nil {
			m.skills = m.skills.SetError(msg.Name, msg.Err)
			m.chat = m.chat.UpdateFunctionResult(msg.Name, fmt.Sprintf("Error: %v", msg.Err))
		} else {
			m.skills = m.skills.SetCompleted(msg.Name)
			m.chat = m.chat.UpdateFunctionResult(msg.Name, msg.Result)

			// Handle NSFW mode toggle
			if msg.Name == "nsfw_mode" && strings.Contains(msg.Result, "enabled") {
				m.nsfwMode = true
				m.header = m.header.SetNSFWMode(true)
			} else if msg.Name == "nsfw_mode" && strings.Contains(msg.Result, "disabled") {
				m.nsfwMode = false
				m.header = m.header.SetNSFWMode(false)
			}
		}

	case SimulateTypingMsg:
		// For simulated streaming (when endpoint dumps all at once)
		displayed := msg.Content[:msg.CharsToShow]
		m.chat = m.chat.SetLastAssistantContent(displayed)
		if msg.CharsToShow < len(msg.Content) {
			// Schedule next typing tick
			cmds = append(cmds, Tick(typingDelay))
		} else {
			m.streaming = false
			m.status = m.status.SetStreaming(false)
		}

	case TickMsg:
		// Handle animation ticks
		var cmd tea.Cmd
		m.status, cmd = m.status.Update(msg)
		cmds = append(cmds, cmd)

	case NSFWToggleMsg:
		m.nsfwMode = msg.Enabled
		m.header = m.header.SetNSFWMode(msg.Enabled)

	case ErrorMsg:
		m.status = m.status.SetText(fmt.Sprintf("Error: %v", msg.Err))
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m AppModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Build the layout vertically
	var sections []string

	// Header (fixed, 1 line)
	sections = append(sections, m.header.View())

	// Chat panel (flexible height)
	sections = append(sections, m.chat.View())

	// Input panel (fixed, 3 lines)
	sections = append(sections, m.input.View())

	// Skills panel (fixed, 5 lines)
	sections = append(sections, m.skills.View())

	// Status bar (fixed, 1 line)
	sections = append(sections, m.status.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// SetLLMClient sets the LLM client.
func (m AppModel) SetLLMClient(client LLMClient) AppModel {
	m.llmClient = client
	if client != nil {
		m.skills = NewSkillsModel(client.GetSkills())
	}
	return m
}

// --- Header Model ---

// HeaderModel represents the header bar.
type HeaderModel struct {
	width    int
	nsfwMode bool
}

// NewHeaderModel creates a new header model.
func NewHeaderModel() HeaderModel {
	return HeaderModel{}
}

// SetWidth sets the header width.
func (m HeaderModel) SetWidth(width int) HeaderModel {
	m.width = width
	return m
}

// SetNSFWMode sets the NSFW mode indicator.
func (m HeaderModel) SetNSFWMode(enabled bool) HeaderModel {
	m.nsfwMode = enabled
	return m
}

// View renders the header.
func (m HeaderModel) View() string {
	title := HeaderTitleStyle.Render("✨ Celeste CLI")
	
	info := HeaderInfoStyle.Render("Press Ctrl+C to exit")
	if m.nsfwMode {
		info = NSFWStyle.Render("[NSFW] ") + info
	}

	// Calculate gap
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(info) - 2
	if gap < 1 {
		gap = 1
	}
	spacer := strings.Repeat("─", gap)

	return HeaderStyle.Width(m.width).Render(
		title + spacer + info,
	)
}

// --- Status Model ---

// StatusModel represents the status bar.
type StatusModel struct {
	width     int
	text      string
	streaming bool
	frame     int
}

// NewStatusModel creates a new status model.
func NewStatusModel() StatusModel {
	return StatusModel{text: "Ready"}
}

// SetWidth sets the status bar width.
func (m StatusModel) SetWidth(width int) StatusModel {
	m.width = width
	return m
}

// SetText sets the status text.
func (m StatusModel) SetText(text string) StatusModel {
	m.text = text
	return m
}

// SetStreaming sets the streaming indicator.
func (m StatusModel) SetStreaming(streaming bool) StatusModel {
	m.streaming = streaming
	return m
}

// Update handles tick messages for animation.
func (m StatusModel) Update(msg tea.Msg) (StatusModel, tea.Cmd) {
	if _, ok := msg.(TickMsg); ok {
		m.frame++
	}
	return m, nil
}

// View renders the status bar.
func (m StatusModel) View() string {
	var status string
	if m.streaming {
		// Animated spinner
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner := StatusStreamingStyle.Render(frames[m.frame%len(frames)])
		status = spinner + " " + StatusStreamingStyle.Render("Streaming...")
	} else {
		status = StatusActiveStyle.Render("●") + " " + m.text
	}

	return StatusBarStyle.Width(m.width).Render(status)
}

// --- Helper functions ---

func helpText() string {
	return `
Commands:
  help    - Show this help
  clear   - Clear chat history
  exit    - Exit the application
  quit    - Exit the application
  
Keyboard shortcuts:
  Ctrl+C     - Exit immediately
  PgUp/PgDn  - Scroll chat history
  Shift+↑/↓  - Scroll chat history
  ↑/↓        - Navigate input history
`
}

// Run starts the TUI application.
func Run(llmClient LLMClient) error {
	p := tea.NewProgram(
		NewApp(llmClient),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}

// Typing delay for simulated streaming (40 chars/sec = 25ms per char)
const typingDelay = 25 * 1000000 // 25ms in nanoseconds

