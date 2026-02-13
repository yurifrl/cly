package agents

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// ReconcileResult summarizes what happened during a sync.
type ReconcileResult struct {
	Written int
	Skipped int
	Errors  []error
}

// Reconcile executes a SyncPlan: reads source, applies transforms, compares, writes.
func Reconcile(plan *SyncPlan, dryRun bool) (*ReconcileResult, error) {
	result := &ReconcileResult{}

	for _, item := range plan.Items {
		written, err := reconcileItem(item, dryRun)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", item.Source, err))
			continue
		}
		if written {
			result.Written++
		} else {
			result.Skipped++
		}
	}

	return result, nil
}

func reconcileItem(item SyncItem, dryRun bool) (bool, error) {
	srcData, err := os.ReadFile(item.Source)
	if err != nil {
		return false, err
	}

	// Apply transform
	outData, err := applyTransform(srcData, item.Transform)
	if err != nil {
		return false, err
	}

	// Compare with existing target
	existing, err := os.ReadFile(item.Target)
	if err == nil && bytes.Equal(existing, outData) {
		return false, nil // unchanged
	}

	if dryRun {
		return true, nil
	}

	// Ensure parent dir
	if err := os.MkdirAll(filepath.Dir(item.Target), 0755); err != nil {
		return false, err
	}

	// Write
	if err := os.WriteFile(item.Target, outData, 0644); err != nil {
		return false, err
	}

	return true, nil
}

func applyTransform(data []byte, kind TransformKind) ([]byte, error) {
	switch kind {
	case TransformJSONCSK:
		return TransformJSONC(data)
	case TransformSkillMD:
		return StripAllowedTools(data), nil
	default:
		return data, nil
	}
}
