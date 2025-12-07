package regulation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type DescriptionGenerator struct {
	chatModel   model.ToolCallingChatModel
	extraPrompt string
}

func NewDescriptionGenerator(
	chatModel model.ToolCallingChatModel,
	extraPrompt string,
) *DescriptionGenerator {
	return &DescriptionGenerator{
		chatModel:   chatModel,
		extraPrompt: extraPrompt,
	}
}

func (g *DescriptionGenerator) Run(ctx context.Context, fileContent string) (string, error) {
	messages := []*schema.Message{
		schema.SystemMessage("You are a high skilled programmer."),
		schema.UserMessage(fmt.Sprintf(
			"Briefly describe what this code does in a single line. Do not include any code. "+
				"Do not include any explanations. Only include the description. "+
				"Don't start with \"The code is\", start directly with what it does. "+
				"Do not use specific programming language keywords. "+
				"Be directly, avoids things like \"Implements a component to display\", just say \"Displays\". "+
				g.extraPrompt+
				"Use the following content as the input:\n%s",
			fileContent,
		)),
	}

	modelCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := g.chatModel.Generate(modelCtx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate completion: %w", err)
	}

	return normalize(response.Content), nil
}

func normalize(description string) string {
	description = strings.ReplaceAll(description, "\n", " ")
	description = strings.TrimSpace(description)

	for strings.Contains(description, "  ") {
		description = strings.ReplaceAll(description, "  ", " ")
	}

	return description
}
