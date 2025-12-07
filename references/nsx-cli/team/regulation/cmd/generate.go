package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/interact"
	regulation "github.com/NSXBet/nsx-cli/team/regulation/internal/generator"
)

var (
	regulationDir    string
	regulationExt    []string
	regulationMarker string
	dryRun           bool
	apiKey           string
	model            string
	prompt           string
	parallelWorkers  int
)

var generateCmd = &cobra.Command{
	Use:   "generate [directory]",
	Short: "Generate descriptions for files with regulation markers",
	Long: `Traverse a repository and generate AI descriptions for files containing
the [--REGULATION_HASHABLE--] marker. Descriptions are added to the marker line.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := regulationDir
		if len(args) > 0 {
			dir = args[0]
		}
		dir = resolveTildeDirectory(dir)

		if dir == "" {
			dir = "."
		}

		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" && !dryRun {
			return fmt.Errorf("OpenAI API key not set. Use --api-key flag or set OPENAI_API_KEY environment variable")
		}

		interact.Info("Starting regulation generation in directory: %s", dir)
		if dryRun {
			interact.Info("Running in dry-run mode (no files will be modified)")
		}

		ctx := context.Background()

		chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey: apiKey,
			Model:  model,
		})
		if err != nil {
			return fmt.Errorf("failed to create OpenAI chat model: %w", err)
		}

		descriptionGenerator := regulation.NewDescriptionGenerator(chatModel, prompt)
		generator := regulation.NewRegulationGenerator(&regulation.RegulationGeneratorConfig{
			ParallelWorkers: parallelWorkers,
			Extensions:      regulationExt,
			Marker:          regulationMarker,
			DryRun:          dryRun,
		}, descriptionGenerator)

		return generator.Run(ctx, dir)
	},
}

func init() {
	generateCmd.Flags().StringVar(&regulationDir, "dir", ".", "directory to traverse")
	generateCmd.Flags().
		StringSliceVarP(&regulationExt, "ext", "e", []string{".ts", ".tsx", ".go", ".cs", ".php"}, "file extensions to process")
	generateCmd.Flags().
		StringVarP(&regulationMarker, "marker", "m", "// [--REGULATION_HASHABLE--]", "regulation marker to search for")
	generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")
	generateCmd.Flags().StringVar(&apiKey, "api-key", "", "OpenAI API key (defaults to OPENAI_API_KEY env var)")
	generateCmd.Flags().StringVar(&prompt, "prompt", "", "Extra prompt to add to the generation")
	generateCmd.Flags().StringVar(&model, "model", "gpt-5", "OpenAI model to use for generation")
	generateCmd.Flags().IntVarP(&parallelWorkers, "workers", "w", 5, "number of parallel workers for file processing")

	RootCmd.AddCommand(generateCmd)
}

func resolveTildeDirectory(dir string) string {
	if strings.HasPrefix(dir, "~/") {
		return filepath.Join(os.Getenv("HOME"), strings.TrimPrefix(dir, "~/"))
	}

	return dir
}
