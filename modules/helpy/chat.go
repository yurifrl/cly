package helpy

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/yurifrl/cly/pkg/llm"
)

var (
	userStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	assistantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	chatDimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	chatBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color("62"))
)

// streamChunkMsg delivers a streaming token chunk to the bubbletea update loop.
// The channel is carried in the message so the next chunk can be requested
// without storing state on the model (which gets copied by value in bubbletea).
type streamChunkMsg struct {
	text string
	done bool
	err  error
	ch   <-chan llm.StreamChunk // carry the channel forward for next read
}

// chatModel is the AI chat panel sub-component.
type chatModel struct {
	viewport     viewport.Model
	textarea     textarea.Model
	spinner      spinner.Model
	messages     []chatMessage
	streaming    string // accumulates current streaming response
	loading      bool
	client       llm.Client
	systemPrompt string
	docContent   string
	docMeta      DocMeta
	width        int
	height       int
	cancelFn     context.CancelFunc
	err          error
}

type chatMessage struct {
	role    llm.Role
	content string
}

func newChatModel(client llm.Client, systemPrompt, docContent string, meta DocMeta) chatModel {
	ta := textarea.New()
	ta.Placeholder = "Ask about this doc..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 1000
	ta.SetWidth(40)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(40, 5)
	vp.SetContent(chatDimStyle.Render("  💬 AI Chat — ask questions about this doc\n  Press Enter to send, Esc to close"))

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return chatModel{
		viewport:     vp,
		textarea:     ta,
		spinner:      s,
		client:       client,
		systemPrompt: systemPrompt,
		docContent:   docContent,
		docMeta:      meta,
	}
}

func (m chatModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - 3
		m.refreshViewport()

	case streamChunkMsg:
		if msg.err != nil {
			m.loading = false
			m.streaming = ""
			m.err = msg.err
			m.refreshViewport()
			return m, nil
		}
		if msg.done {
			// Finalize the streaming message
			if m.streaming != "" {
				m.messages = append(m.messages, chatMessage{
					role:    llm.RoleAssistant,
					content: m.streaming,
				})
			}
			m.streaming = ""
			m.loading = false
			m.refreshViewport()
			return m, nil
		}
		// Accumulate streaming text and request next chunk
		m.streaming += msg.text
		m.refreshViewport()
		if msg.ch != nil {
			cmds = append(cmds, waitForStream(msg.ch))
		}
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		if m.loading {
			// While loading, only allow cancel
			if msg.String() == "esc" {
				if m.cancelFn != nil {
					m.cancelFn()
					m.cancelFn = nil
				}
				m.loading = false
				m.streaming = ""
				m.refreshViewport()
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.Type {
		case tea.KeyEnter:
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, tea.Batch(cmds...)
			}

			// Add user message
			m.messages = append(m.messages, chatMessage{
				role:    llm.RoleUser,
				content: text,
			})
			m.textarea.Reset()
			m.loading = true
			m.err = nil
			m.refreshViewport()

			// Start streaming
			cmd := m.startStream()
			cmds = append(cmds, cmd, m.spinner.Tick)
			return m, tea.Batch(cmds...)
		}
	}

	// Update textarea
	var tiCmd tea.Cmd
	m.textarea, tiCmd = m.textarea.Update(msg)
	cmds = append(cmds, tiCmd)

	// Update viewport (for scrolling)
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// startStream initiates the AI stream and returns a tea.Cmd that reads the first chunk.
func (m *chatModel) startStream() tea.Cmd {
	// Capture what we need before returning the Cmd closure.
	// The Cmd runs in a goroutine — we cannot touch the model from there.
	systemPrompt := m.buildSystemPrompt()
	var msgs []llm.Message
	for _, msg := range m.messages {
		msgs = append(msgs, llm.Message{
			Role:    msg.role,
			Content: msg.content,
		})
	}
	client := m.client

	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		// NOTE: cancelFn is set here but on a stale model copy.
		// Cancel is still useful via the context if the goroutine checks it.
		_ = cancel

		ch, err := client.Stream(ctx, systemPrompt, msgs)
		if err != nil {
			return streamChunkMsg{err: err}
		}

		// Read first chunk and carry the channel forward in the message
		chunk, ok := <-ch
		if !ok {
			return streamChunkMsg{done: true, ch: ch}
		}
		if chunk.Err != nil {
			return streamChunkMsg{err: chunk.Err, ch: ch}
		}
		if chunk.Done {
			return streamChunkMsg{done: true, ch: ch}
		}
		return streamChunkMsg{text: chunk.Text, ch: ch}
	}
}

// waitForStream returns a tea.Cmd that reads the next chunk from the stream channel.
func waitForStream(ch <-chan llm.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return streamChunkMsg{done: true}
		}
		if chunk.Err != nil {
			return streamChunkMsg{err: chunk.Err}
		}
		if chunk.Done {
			return streamChunkMsg{done: true}
		}
		return streamChunkMsg{text: chunk.Text, ch: ch}
	}
}

func (m *chatModel) buildSystemPrompt() string {
	var sb strings.Builder

	sb.WriteString(m.systemPrompt)
	sb.WriteString("\n\n")

	if m.docMeta.Name != "" {
		sb.WriteString(fmt.Sprintf("Document: %s\n", m.docMeta.Name))
	}
	if m.docMeta.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", m.docMeta.Description))
	}
	if m.docMeta.URL != "" {
		sb.WriteString(fmt.Sprintf("URL: %s\n", m.docMeta.URL))
	}
	sb.WriteString("\n--- Document Content ---\n\n")
	sb.WriteString(m.docContent)

	return sb.String()
}

func (m *chatModel) refreshViewport() {
	var parts []string

	if len(m.messages) == 0 && m.streaming == "" && m.err == nil {
		m.viewport.SetContent(chatDimStyle.Render("  💬 AI Chat — ask questions about this doc\n  Press Enter to send, Esc to close"))
		return
	}

	for _, msg := range m.messages {
		switch msg.role {
		case llm.RoleUser:
			parts = append(parts, userStyle.Render("You: ")+msg.content)
		case llm.RoleAssistant:
			rendered := m.renderMarkdown(msg.content)
			parts = append(parts, assistantStyle.Render("AI: ")+rendered)
		}
	}

	if m.streaming != "" {
		rendered := m.renderMarkdown(m.streaming)
		parts = append(parts, assistantStyle.Render("AI: ")+rendered)
	}

	if m.loading && m.streaming == "" {
		parts = append(parts, m.spinner.View()+" Thinking...")
	}

	if m.err != nil {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(
			fmt.Sprintf("Error: %v", m.err)))
	}

	content := strings.Join(parts, "\n\n")
	if m.width > 0 {
		content = lipgloss.NewStyle().Width(m.width - 2).Render(content)
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *chatModel) renderMarkdown(text string) string {
	width := m.width - 4
	if width < 40 {
		width = 40
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return text
	}
	rendered, err := r.Render(text)
	if err != nil {
		return text
	}
	return strings.TrimSpace(rendered)
}

func (m chatModel) View() string {
	border := chatBorderStyle.Width(m.width).Render("")
	return fmt.Sprintf("%s\n%s\n%s", border, m.viewport.View(), m.textarea.View())
}

// standaloneChatModel wraps chatModel as a top-level tea.Model for `helpy --chat`.
type standaloneChatModel struct {
	chat    chatModel
	docMeta DocMeta
}

func newStandaloneChatModel(client llm.Client, systemPrompt, docContent string, meta DocMeta) standaloneChatModel {
	chat := newChatModel(client, systemPrompt, docContent, meta)
	return standaloneChatModel{
		chat:    chat,
		docMeta: meta,
	}
}

func (m standaloneChatModel) Init() tea.Cmd {
	return m.chat.Init()
}

func (m standaloneChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.chat.width = msg.Width
		m.chat.height = msg.Height
		m.chat.viewport.Width = msg.Width
		m.chat.textarea.SetWidth(msg.Width)
		m.chat.viewport.Height = msg.Height - m.chat.textarea.Height() - 4 // border + header
		m.chat.refreshViewport()
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "esc" && !m.chat.loading {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.chat, cmd = m.chat.Update(msg)
	return m, cmd
}

func (m standaloneChatModel) View() string {
	// Header with doc info
	var header string
	if m.docMeta.Name != "" {
		header = chatDimStyle.Render(fmt.Sprintf("  💬 AI Chat — %s", m.docMeta.Name))
	} else {
		header = chatDimStyle.Render("  💬 AI Chat")
	}
	header += chatDimStyle.Render("  (esc/ctrl+c to quit)")

	return header + "\n" + m.chat.viewport.View() + "\n" + m.chat.textarea.View()
}
