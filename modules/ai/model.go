package ai

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Message represents a chat message
type Message struct {
	Role    string
	Content string
	Time    time.Time
}

// Model is the Bubbletea model for the AI chat
type Model struct {
	// UI Components
	textarea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	// State
	conversationID string
	messages       []Message
	width          int
	height         int
	loading        bool

	// Client
	client *Client
	model  string

	err      error
	quitting bool
}

// NewModel creates a new AI chat model
func NewModel(apiKey, model string) (Model, error) {
	// Create client
	client, err := NewClient(apiKey, model)
	if err != nil {
		return Model{}, fmt.Errorf("failed to create client: %w", err)
	}

	// Initialize textarea
	ti := textarea.New()
	ti.Placeholder = "Type your message... (Ctrl+Enter to send, Esc to quit)"
	ti.Focus()
	ti.CharLimit = 0
	ti.SetHeight(3)

	// Initialize viewport
	vp := viewport.New(80, 20)

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	// Generate conversation ID
	conversationID := generateConversationID()

	return Model{
		textarea:       ti,
		viewport:       vp,
		spinner:        sp,
		conversationID: conversationID,
		messages:       []Message{},
		client:         client,
		model:          model,
		loading:        false,
		quitting:       false,
	}, nil
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	return fmt.Sprintf("modsi-%d", time.Now().Unix())
}
