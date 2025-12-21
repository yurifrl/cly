package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yurifrl/cly/modules/scraper/aliexpress"
)

// OutputMode defines how output is written
type OutputMode int

const (
	// SingleFile writes all products to one JSON array
	SingleFile OutputMode = iota
	// PerURL writes each product to separate file
	PerURL
)

// Writer handles writing product data to files
type Writer struct {
	mode      OutputMode
	outputDir string
	file      *os.File
	isFirst   bool
}

// NewWriter creates a new output writer
func NewWriter(mode OutputMode, outputPath string) (*Writer, error) {
	w := &Writer{
		mode:      mode,
		outputDir: filepath.Dir(outputPath),
		isFirst:   true,
	}

	// Create output directory
	if err := os.MkdirAll(w.outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	// For single file mode, open file and write opening bracket
	if mode == SingleFile {
		f, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		w.file = f
		if _, err := f.WriteString("[\n"); err != nil {
			return nil, err
		}
	}

	return w, nil
}

// WriteProduct writes a product to output
func (w *Writer) WriteProduct(product *aliexpress.ProductData) error {
	if w.mode == PerURL {
		return w.writeToSeparateFile(product)
	}
	return w.appendToSingleFile(product)
}

// writeToSeparateFile writes product to its own file
func (w *Writer) writeToSeparateFile(product *aliexpress.ProductData) error {
	filename := filepath.Join(w.outputDir, fmt.Sprintf("product_%d.json", product.ProductID))

	data, err := json.MarshalIndent(product, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// appendToSingleFile appends product to JSON array
func (w *Writer) appendToSingleFile(product *aliexpress.ProductData) error {
	if w.file == nil {
		return fmt.Errorf("output file not open")
	}

	// Write comma if not first entry
	if !w.isFirst {
		if _, err := w.file.WriteString(",\n"); err != nil {
			return err
		}
	}
	w.isFirst = false

	// Marshal with indentation
	data, err := json.MarshalIndent(product, "  ", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	if _, err := w.file.Write(data); err != nil {
		return err
	}

	return nil
}

// Close closes the writer and finalizes output
func (w *Writer) Close() error {
	if w.file != nil {
		// Write closing bracket
		if _, err := w.file.WriteString("\n]\n"); err != nil {
			return err
		}
		return w.file.Close()
	}
	return nil
}
