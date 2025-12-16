package ai

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Custom message types
type apiResponseMsg struct {
	response string
	err      error
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyCtrlJ: // Ctrl+Enter sends message
			if m.loading {
				return m, nil
			}

			userMsg := m.textarea.Value()
			if userMsg == "" {
				return m, nil
			}

			// Add user message to history
			m.messages = append(m.messages, Message{
				Role:    "user",
				Content: userMsg,
				Time:    time.Now(),
			})

			// Clear input
			m.textarea.Reset()

			// Start loading
			m.loading = true

			// Update viewport
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()

			return m, tea.Batch(
				m.sendMessage(userMsg),
				m.spinner.Tick,
			)
		}

	case apiResponseMsg:
		m.loading = false

		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// Add assistant response
		m.messages = append(m.messages, Message{
			Role:    "assistant",
			Content: msg.response,
			Time:    time.Now(),
		})

		// Update viewport
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Adjust component sizes
		m.textarea.SetWidth(msg.Width - 4)
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10

		return m, nil

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Update textarea
	if !m.loading {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// sendMessage sends a message via mods (async)
func (m Model) sendMessage(userMsg string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		response, err := m.client.SendMessage(ctx, m.conversationID, userMsg)

		return apiResponseMsg{
			response: response,
			err:      err,
		}
	}
}
