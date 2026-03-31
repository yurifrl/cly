package oi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/yurifrl/cly/pkg/llm"
)

var (
	// Styles
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).PaddingLeft(1)
	langTag     = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	okMark      = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))  // green
	errMark     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // red
	dimText     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingLeft(3)
	accentText  = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).PaddingLeft(3)
	plainWord   = lipgloss.NewStyle() // NO formatting — for easy copy
	boxBorder   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)
	posStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Italic(true).PaddingLeft(3) // part of speech
	defStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).PaddingLeft(5)              // definition
	exStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true).PaddingLeft(5) // example
	pronStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).PaddingLeft(1)              // pronunciation
	tipStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).PaddingLeft(3)              // mnemonic/tip
	candNum   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).PaddingLeft(3)              // candidate number
	candWord  = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)                  // candidate word
	candDef   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))                             // candidate definition
)

// LLM response structure
type spellResponse struct {
	Input         string       `json:"input"`
	Language      string       `json:"language"`
	IsCorrect     bool         `json:"is_correct"`
	Corrected     string       `json:"corrected"`
	IsWord        bool         `json:"is_word"`
	Pronunciation string       `json:"pronunciation,omitempty"`
	PartOfSpeech  string       `json:"part_of_speech,omitempty"`
	Definition    string       `json:"definition,omitempty"`
	Example       string       `json:"example,omitempty"`
	Explanation   string       `json:"explanation,omitempty"`
	Mnemonic      string       `json:"mnemonic,omitempty"`
	Candidates    []candidate  `json:"candidates,omitempty"`
	Changes       []wordChange `json:"changes,omitempty"`
}

type candidate struct {
	Word       string `json:"word"`
	Definition string `json:"definition"`
}

type wordChange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

const systemPrompt = `You are a spelling checker for English and Brazilian Portuguese (pt-BR) ONLY.

Given input text, determine:
1. Whether it's a single word or a phrase
2. The language (English or pt-BR)
3. Whether spelling is correct
4. The corrected version if wrong

For SINGLE WORDS, also provide:
- pronunciation (IPA or phonetic)
- part of speech
- brief definition in the SAME language as the word
- usage example
- if misspelled: explanation of the mistake and a mnemonic tip to remember
- CRITICAL: when misspelled, provide 2-5 candidate words the user might have meant in "candidates". You MUST consider candidates from BOTH languages (English AND pt-BR). Consider phonetic similarity, common typo patterns, letter transpositions, and similar-sounding words across both languages. The first candidate should be your best guess (same as "corrected"), but ALWAYS include alternatives from the other language too. Example: "lachante" could be "laxante" (pt-BR, laxative), "laughing" (English), etc. People often know roughly what word they want but mangle the spelling — give them options from both languages. Include the language in each candidate definition, e.g. "pt-BR: substância que facilita a evacuação".

For PHRASES:
- if the user seems to be asking about a specific word (e.g. "how do you spell X", "is X correct"), treat it as a word query for X
- otherwise, correct the entire phrase and list each change

ALWAYS respond with valid JSON matching this schema:
{
  "input": "original input",
  "language": "English" or "pt-BR",
  "is_correct": true/false,
  "corrected": "most likely correction",
  "is_word": true/false,
  "pronunciation": "/IPA/ or phonetic (words only)",
  "part_of_speech": "noun/verb/adj/etc (words only)",
  "definition": "brief definition in same language (words only)",
  "example": "usage example sentence (words only)",
  "explanation": "why the spelling is wrong (if misspelled)",
  "mnemonic": "memory tip to remember correct spelling (if misspelled)",
  "candidates": [{"word": "candidate1", "definition": "brief definition"}, ...],
  "changes": [{"from": "wrong", "to": "right"}]
}

Respond ONLY with JSON. No markdown, no code fences, no extra text.`

func newLLMClient() (llm.Client, error) {
	return llm.NewClient(llm.Config{
		Provider: llm.ProviderAnthropic,
		Model:    "claude-sonnet-4-20250514",
	})
}

func check(input string) error {
	client, err := newLLMClient()
	if err != nil {
		return fmt.Errorf("LLM client error: %w", err)
	}

	ctx := context.Background()
	resp, err := client.Complete(ctx, systemPrompt, []llm.Message{
		{Role: llm.RoleUser, Content: input},
	})
	if err != nil {
		return fmt.Errorf("LLM error: %w", err)
	}

	// Parse JSON response
	resp = strings.TrimSpace(resp)
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var result spellResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return fmt.Errorf("failed to parse LLM response: %w\nraw: %s", err, resp)
	}

	return render(result)
}

func render(r spellResponse) error {
	// Verbosity 0: just the corrected text, nothing else
	if verbosity == 0 {
		fmt.Println(r.Corrected)
		return nil
	}

	var out strings.Builder

	if r.IsWord {
		renderWord(&out, r)
	} else {
		renderPhrase(&out, r)
	}

	fmt.Print(boxBorder.Render(out.String()))
	fmt.Println()
	return nil
}

func renderWord(out *strings.Builder, r spellResponse) {
	// Header: word + pronunciation + language
	header := headerStyle.Render(r.Input)
	if r.Pronunciation != "" {
		header += " " + pronStyle.Render(r.Pronunciation)
	}
	header += " " + langTag.Render("("+r.Language+")")
	if r.IsCorrect {
		header += " " + okMark.Render("✓")
	} else {
		header += " " + errMark.Render("✗")
	}
	out.WriteString(header + "\n")

	// If misspelled, show corrected word PLAIN (no formatting) for copy
	if !r.IsCorrect {
		out.WriteString("\n")
		out.WriteString(plainWord.Render(r.Corrected) + "\n")
	}

	// Part of speech + definition
	if r.PartOfSpeech != "" {
		out.WriteString("\n")
		out.WriteString(posStyle.Render(r.PartOfSpeech) + "\n")
	}
	if r.Definition != "" {
		out.WriteString(defStyle.Render(r.Definition) + "\n")
	}

	// Show other candidates ("did you mean...")
	if !r.IsCorrect && len(r.Candidates) > 1 {
		out.WriteString("\n" + dimText.Render("did you mean:") + "\n")
		for i, c := range r.Candidates {
			num := fmt.Sprintf("%d.", i+1)
			line := candNum.Render(num) + " " + candWord.Render(c.Word)
			if c.Definition != "" {
				line += " " + candDef.Render("— "+c.Definition)
			}
			out.WriteString(line + "\n")
		}
	}

	// Verbosity 2: example, explanation, mnemonic
	if verbosity >= 2 {
		if r.Example != "" {
			out.WriteString(exStyle.Render("\""+r.Example+"\"") + "\n")
		}
		if r.Explanation != "" {
			out.WriteString("\n" + dimText.Render(r.Explanation) + "\n")
		}
		if r.Mnemonic != "" {
			out.WriteString(tipStyle.Render("💡 "+r.Mnemonic) + "\n")
		}
	} else if verbosity >= 1 && !r.IsCorrect {
		// V1: show explanation but not example/mnemonic
		if r.Explanation != "" {
			out.WriteString("\n" + dimText.Render(r.Explanation) + "\n")
		}
	}
}

func renderPhrase(out *strings.Builder, r spellResponse) {
	header := langTag.Render("(" + r.Language + ")")
	if r.IsCorrect {
		header += " " + okMark.Render("✓ correct")
	} else {
		header += " " + errMark.Render("✗")
	}
	out.WriteString(header + "\n\n")

	// Corrected phrase PLAIN for copy
	out.WriteString(plainWord.Render(r.Corrected) + "\n")

	if !r.IsCorrect && verbosity >= 1 {
		if len(r.Changes) > 0 {
			out.WriteString("\n" + dimText.Render("changes:") + "\n")
			for _, c := range r.Changes {
				out.WriteString(accentText.Render(
					errMark.Render(c.From)+" → "+okMark.Render(c.To),
				) + "\n")
			}
		}
	}

	if verbosity >= 2 && r.Explanation != "" {
		out.WriteString("\n" + dimText.Render(r.Explanation) + "\n")
	}
}

// checkForInteractive is used by the TUI — returns formatted string instead of printing
func checkForInteractive(input string) string {
	client, err := newLLMClient()
	if err != nil {
		return errMark.Render("error: " + err.Error())
	}

	ctx := context.Background()
	resp, err := client.Complete(ctx, systemPrompt, []llm.Message{
		{Role: llm.RoleUser, Content: input},
	})
	if err != nil {
		return errMark.Render("error: " + err.Error())
	}

	resp = strings.TrimSpace(resp)
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var result spellResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return errMark.Render("parse error: " + err.Error())
	}

	var out strings.Builder
	if result.IsWord {
		renderWord(&out, result)
	} else {
		renderPhrase(&out, result)
	}
	return boxBorder.Render(out.String()) + "\n"
}
