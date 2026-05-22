package gitcommits

import (
	"fmt"
	"strings"
)

// defaultBatchSize is the per-batch character budget sent to the LLM.
// Sized for modern context windows so most changesets fit in a single batch,
// avoiding fragmentation across independent planning calls.
const defaultBatchSize = 150000

// Batch represents a subset of files to send to the LLM in one request.
type Batch struct {
	Index      int
	TotalCount int
	Files      []FileChange
	Text       string // The analysis text sent to the LLM
}

// BuildBatches splits a changeset into batches that fit within the character budget.
func BuildBatches(cs *Changeset, batchSize int) []Batch {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	type fileText struct {
		file FileChange
		text string
		size int
	}

	// Generate analysis text for each file
	var items []fileText
	for _, f := range cs.Files {
		text := buildFileAnalysis(f)
		items = append(items, fileText{
			file: f,
			text: text,
			size: len(text),
		})
	}

	// Greedy packing by character budget
	var batches []Batch
	var currentFiles []FileChange
	var currentTexts []string
	currentSize := 0

	for _, item := range items {
		// If a single file exceeds budget, it gets its own batch
		if currentSize+item.size > batchSize && len(currentFiles) > 0 {
			batches = append(batches, makeBatch(len(batches), currentFiles, currentTexts))
			currentFiles = nil
			currentTexts = nil
			currentSize = 0
		}
		currentFiles = append(currentFiles, item.file)
		currentTexts = append(currentTexts, item.text)
		currentSize += item.size
	}

	// Don't forget the last batch
	if len(currentFiles) > 0 {
		batches = append(batches, makeBatch(len(batches), currentFiles, currentTexts))
	}

	// Set total count on all batches
	total := len(batches)
	for i := range batches {
		batches[i].TotalCount = total
	}

	return batches
}

func makeBatch(index int, files []FileChange, texts []string) Batch {
	return Batch{
		Index: index,
		Files: files,
		Text:  strings.Join(texts, "\n---\n"),
	}
}

// buildFileAnalysis generates the analysis text for a single file.
func buildFileAnalysis(f FileChange) string {
	var b strings.Builder

	// File header
	status := string(f.Status)
	switch f.Status {
	case StatusAdded:
		status = "ADDED"
	case StatusModified:
		status = "MODIFIED"
	case StatusDeleted:
		status = "DELETED"
	case StatusRenamed:
		status = "RENAMED"
	}

	b.WriteString(fmt.Sprintf("File: %s [%s]\n", f.Path, status))

	if f.OldPath != "" {
		b.WriteString(fmt.Sprintf("  Renamed from: %s\n", f.OldPath))
	}

	// Hunk summaries
	for _, h := range f.Hunks {
		b.WriteString(fmt.Sprintf("  Hunk %s: %s\n", h.ID, h.RangeLabel))
	}

	// Skip diff for binary files (GIT binary patch, "Binary files differ")
	isBinary := strings.Contains(f.Diff, "GIT binary patch") ||
		strings.Contains(f.Diff, "Binary files")

	// Include diff if it's text and not too large (32K secondary limit per file)
	if !isBinary && len(f.Diff) <= 32000 {
		b.WriteString("\n```diff\n")
		b.WriteString(f.Diff)
		if !strings.HasSuffix(f.Diff, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n")
	}

	return b.String()
}
