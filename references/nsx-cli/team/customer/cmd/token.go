package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/interact"
	"github.com/NSXBet/nsx-cli/shared/skin"
	"github.com/NSXBet/nsx-cli/team/customer/customercfg"
	"github.com/NSXBet/nsx-cli/team/customer/internal/api"
)

const (
	tokenSecretKey = "JWT_SECRET"
)

var (
	focusedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	noStyle          = lipgloss.NewStyle()
	generateTokenCmd = &cobra.Command{
		Use:   "generate-token",
		Short: "Generate a new token for the customer",
		Long:  `Generate a new token for the customer. This token can be used for API access or other integrations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := customercfg.GetCustomerServiceConfig()
			if err != nil {
				return fmt.Errorf("failed to get customer service config: %w", err)
			}

			token, ok := cfg[tokenSecretKey]
			if !ok {
				return fmt.Errorf("token not found")
			}

			if token == "" {
				fmt.Println("❌ You must provide a --token argument")
				return fmt.Errorf("token is required")
			}

			interact.Debug("🔑 Token found: %s", token)

			// Start TUI
			p := tea.NewProgram(initialModel(token, cfg))
			if _, err := p.Run(); err != nil {
				fmt.Printf("Error running program: %v\n", err)
				return err
			}
			return nil
		},
	}
)

func init() {
	RootCmd.AddCommand(generateTokenCmd)
}

type model struct {
	cfg        map[string]string
	token      string
	inputs     []textinput.Model
	focusIndex int
	done       bool
	jwtToken   string
	copyError  error
}

func initialModel(token string, cfg map[string]string) model {
	inputs := make([]textinput.Model, 2)

	t1 := textinput.New()
	t1.Prompt = "Customer ID: "
	t1.Placeholder = "12345"
	t1.Width = 50
	t1.Focus()
	t1.PromptStyle = focusedStyle
	t1.TextStyle = focusedStyle

	t2 := textinput.New()
	t2.Prompt = "Duration: "
	t2.Placeholder = "30d (e.g. 30d, 24h, 1h30m)"
	t2.Width = 50
	t2.PromptStyle = noStyle
	t2.TextStyle = noStyle

	inputs[0] = t1
	inputs[1] = t2

	return model{
		cfg:        cfg,
		token:      token,
		inputs:     inputs,
		focusIndex: 0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			if m.focusIndex == len(m.inputs)-1 {
				customerService := api.NewCustomerServiceClient(m.cfg)
				customerData, err := customerService.GetCustomerByID(m.inputs[0].Value())
				if err != nil {
					fmt.Printf("Error getting customer data: %v\n", err)
					return m, tea.Quit
				}
				jwtToken, err := generateJWT(m.token, customerData.ID, m.inputs[1].Value(), customerData.UserID)
				if err != nil {
					fmt.Printf("Error generating JWT: %v\n", err)
					return m, tea.Quit
				}

				interact.Debug("🔑 JWT Token Generated Successfully! %s", jwtToken)
				m.jwtToken = jwtToken

				// Copy to clipboard
				m.copyError = clipboard.WriteAll(jwtToken)

				m.done = true
				return m, tea.Quit
			}
			m.focusIndex++
			for i := range m.inputs {
				if i == m.focusIndex {
					m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
				} else {
					m.inputs[i].Blur()
					m.inputs[i].PromptStyle = noStyle
					m.inputs[i].TextStyle = noStyle
				}
			}
		}
	}

	// Update inputs
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.done {
		// Create a styled box for the token
		tokenStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1).
			Width(80)

		clipboardMsg := "📋 Token copied to clipboard!"
		if m.copyError != nil {
			clipboardMsg = fmt.Sprintf("⚠️  Clipboard copy failed: %v", m.copyError)
		}

		return fmt.Sprintf(
			"\n✅ JWT Token Generated Successfully!\n\n%s\n\n%s\n\n%s\n\n",
			clipboardMsg,
			tokenStyle.Render(m.jwtToken),
			"💡 The token above is also displayed for reference (already in your clipboard)",
		)
	}

	// Create a styled container for the form
	s := "\n🔑 Customer Token Generator\n\n"
	s += "Please provide the following information:\n\n"

	for i := range m.inputs {
		s += m.inputs[i].View() + "\n\n"
	}

	s += "💡 Press Enter to continue to next field, Enter on last field to generate token\n"
	s += "🚪 Press Esc to quit\n"
	return s
}

func generateJWT(secretToken string, customerID int, duration string, userID int) (string, error) {
	now := time.Now()

	// Parse duration
	expDuration, err := time.ParseDuration(duration)
	if err != nil {
		// Try parsing as days (e.g., "30d")
		if len(duration) > 1 && duration[len(duration)-1] == 'd' {
			days, parseErr := strconv.Atoi(duration[:len(duration)-1])
			if parseErr != nil {
				return "", fmt.Errorf("invalid duration format: %s", duration)
			}
			expDuration = time.Duration(days) * 24 * time.Hour
		} else {
			return "", fmt.Errorf("invalid duration format: %s", duration)
		}
	}

	// Create custom field
	customData := map[string]interface{}{
		"CustomerId": customerID,
		"Settings": map[string]interface{}{
			"RegisteredSince": now.Format("2006-01-02"),
			"Email": map[string]interface{}{
				"Domain": "nsx.bet",
			},
		},
	}

	customJSON, err := json.Marshal(customData)
	if err != nil {
		return "", err
	}

	// Create JWT claims
	claims := jwt.MapClaims{
		"jti":    uuid.New().String(),
		"iat":    now.Unix(),
		"sub":    userID,
		"sys":    skin.GetSkin().String(),
		"cid":    customerID,
		"custom": string(customJSON),
		"exp":    now.Add(expDuration).Unix(),
		"iss":    "nsx-cli",
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(secretToken))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
