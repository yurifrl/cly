package regulation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/boyter/gocodewalker"

	"github.com/NSXBet/nsx-cli/shared/filesystem"
	"github.com/NSXBet/nsx-cli/shared/interact"
)

type RegulationGeneratorConfig struct {
	ParallelWorkers int
	Extensions      []string
	Marker          string
	DryRun          bool
}

type RegulationGenerator struct {
	config               *RegulationGeneratorConfig
	descriptionGenerator *DescriptionGenerator
}

type processResult struct {
	filePath string
	changed  bool
	err      error
}

func NewRegulationGenerator(
	config *RegulationGeneratorConfig,
	descriptionGenerator *DescriptionGenerator,
) *RegulationGenerator {
	return &RegulationGenerator{
		config:               config,
		descriptionGenerator: descriptionGenerator,
	}
}

func (g *RegulationGenerator) Run(ctx context.Context, dir string) error {
	fileQueue := make(chan *gocodewalker.File, 1000)
	resultChan := make(chan processResult, 1000)

	fileWalker := gocodewalker.NewFileWalker(dir, fileQueue)
	fileWalker.SetErrorHandler(func(err error) bool {
		interact.Error("Error walking files: %v", err)
		return true
	})

	for _, ext := range g.config.Extensions {
		ext = strings.TrimPrefix(ext, ".")
		fileWalker.AllowListExtensions = append(fileWalker.AllowListExtensions, ext)
	}

	var wg sync.WaitGroup
	for i := 0; i < g.config.ParallelWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileQueue {
				changed, err := g.processFile(ctx, file.Location)
				resultChan <- processResult{
					filePath: file.Location,
					changed:  changed,
					err:      err,
				}
			}
		}()
	}

	go func() {
		if err := fileWalker.Start(); err != nil {
			interact.Error("Error starting file walker: %v", err)
		}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	processed := 0
	skipped := 0
	errors := 0

	for result := range resultChan {
		relPath, err := filepath.Rel(dir, result.filePath)
		if err != nil {
			relPath = result.filePath
		}

		if result.err != nil {
			interact.Error("Error processing file %s: %v", relPath, result.err)
			errors++
			continue
		}

		if result.changed {
			processed++

			if g.config.DryRun {
				interact.Info("Would process: %s", relPath)
			} else {
				interact.Info("Processed: %s", relPath)
			}
		} else {
			skipped++
			interact.Debug("Skipped: %s", relPath)
		}
	}

	interact.Info("Summary:")
	interact.Info("  Processed: %d files", processed)
	interact.Info("  Skipped:   %d files", skipped)

	if errors > 0 {
		interact.Info("  Errors:    %d files", errors)
	}

	return nil
}

func (g *RegulationGenerator) processFile(ctx context.Context, path string) (bool, error) {
	content, err := filesystem.ReadFileAsString(path)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return false, nil
	}

	markerLineIndex := -1
	maxLines := min(len(lines), 5)

	for i := range lines[:maxLines] {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, g.config.Marker) {
			markerLineIndex = i
			break
		}
	}

	if markerLineIndex == -1 {
		return false, nil
	}

	markerLine := strings.TrimSpace(lines[markerLineIndex])
	prefix := g.config.Marker + " - "
	if strings.HasPrefix(markerLine, prefix) {
		description := strings.TrimSpace(markerLine[len(prefix):])
		if description != "" {
			interact.Debug("File already has description: %s", path)
			return false, nil
		}
	}

	if g.config.DryRun {
		return true, nil
	}

	description, err := g.descriptionGenerator.Run(ctx, content)
	if err != nil {
		return false, fmt.Errorf("failed to generate description: %w", err)
	}

	lines[markerLineIndex] = fmt.Sprintf("%s - %s", g.config.Marker, description)
	newContent := strings.Join(lines, "\n")

	err = os.WriteFile(path, []byte(newContent), filesystem.FilePermission)
	if err != nil {
		return false, fmt.Errorf("failed to write file: %w", err)
	}

	return true, nil
}
