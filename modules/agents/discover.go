package agents

import (
	"os"
	"path/filepath"
	"strings"
)

// SyncItem represents a single file to sync from source to target.
type SyncItem struct {
	Source        string
	Target        string
	Transform     TransformKind
	Bidirectional bool // true if target edits should sync back to source
}

// SyncPlan is the list of items to sync for one IDE.
type SyncPlan struct {
	Items      []SyncItem
	ReverseMap map[string]string // target path → source path (bidirectional items only)
}

// Discover walks the source .agents directory and produces a SyncPlan for the given IDE.
func Discover(sourceDir string, ide *IDEDef, targetBase string) (*SyncPlan, error) {
	plan := &SyncPlan{
		ReverseMap: make(map[string]string),
	}

	// Phase 1: Shared configs (root of .agents)
	if err := discoverShared(sourceDir, ide, targetBase, plan); err != nil {
		return nil, err
	}

	// Phase 2: IDE-specific overrides from .agents/ides/<ide>/
	ideDir := filepath.Join(sourceDir, "ides", ide.Name)
	if err := discoverIDESpecific(ideDir, ide, targetBase, plan); err != nil {
		return nil, err
	}

	return plan, nil
}

func addItem(plan *SyncPlan, item SyncItem) {
	item.Bidirectional = item.Transform == TransformNone
	plan.Items = append(plan.Items, item)
	if item.Bidirectional {
		plan.ReverseMap[item.Target] = item.Source
	}
}

// discoverShared handles commands/, agents/, skills/, AGENTS.md, and special files.
func discoverShared(sourceDir string, ide *IDEDef, targetBase string, plan *SyncPlan) error {
	// Subdirectories
	for _, subdir := range Subdirs {
		srcSub := filepath.Join(sourceDir, subdir)
		if _, err := os.Stat(srcSub); os.IsNotExist(err) {
			continue
		}

		targetSubdir := subdir
		if renamed, ok := ide.DirRenames[subdir]; ok {
			targetSubdir = renamed
		}

		err := filepath.Walk(srcSub, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			if !shouldSyncFile(path) {
				return nil
			}

			rel, _ := filepath.Rel(srcSub, path)
			targetPath := filepath.Join(targetBase, targetSubdir, rel)
			transform := classifyTransform(path, ide)

			// JSONC files get .json extension in target
			if strings.HasSuffix(targetPath, ".jsonc") && transform == TransformJSONCSK {
				targetPath = strings.TrimSuffix(targetPath, ".jsonc") + ".json"
			}

			addItem(plan, SyncItem{
				Source:    path,
				Target:    targetPath,
				Transform: transform,
			})
			return nil
		})
		if err != nil {
			return err
		}
	}

	// AGENTS.md → renamed target
	agentsMD := filepath.Join(sourceDir, "AGENTS.md")
	if _, err := os.Stat(agentsMD); err == nil {
		addItem(plan, SyncItem{
			Source:    agentsMD,
			Target:    filepath.Join(targetBase, ide.AgentsMDTarget),
			Transform: TransformNone,
		})
	}

	// Special files (e.g. claude.json → settings.json)
	for srcName, dstName := range ide.SpecialFiles {
		srcPath := filepath.Join(sourceDir, srcName)
		if _, err := os.Stat(srcPath); err == nil {
			if !shouldSyncFile(srcPath) {
				continue
			}
			transform := TransformNone
			if strings.HasSuffix(srcName, ".jsonc") {
				transform = TransformJSONCSK
			}
			addItem(plan, SyncItem{
				Source:    srcPath,
				Target:    filepath.Join(targetBase, dstName),
				Transform: transform,
			})
		}
	}

	return nil
}

// discoverIDESpecific handles files from .agents/ides/<ide>/ — copied as-is with transforms.
func discoverIDESpecific(ideDir string, ide *IDEDef, targetBase string, plan *SyncPlan) error {
	if _, err := os.Stat(ideDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(ideDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !shouldSyncFile(path) {
			return nil
		}

		rel, _ := filepath.Rel(ideDir, path)
		targetPath := filepath.Join(targetBase, rel)
		transform := classifyTransform(path, ide)

		if strings.HasSuffix(targetPath, ".jsonc") && transform == TransformJSONCSK {
			targetPath = strings.TrimSuffix(targetPath, ".jsonc") + ".json"
		}

		addItem(plan, SyncItem{
			Source:    path,
			Target:    targetPath,
			Transform: transform,
		})
		return nil
	})
}

// classifyTransform determines what transform to apply to a file.
func classifyTransform(path string, ide *IDEDef) TransformKind {
	base := filepath.Base(path)

	if strings.HasSuffix(base, ".jsonc") {
		return TransformJSONCSK
	}

	if base == "SKILL.md" && ide.StripAllowedTools {
		return TransformSkillMD
	}

	return TransformNone
}

func shouldSyncFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".yaml", ".yml", ".json", ".jsonc":
		return true
	default:
		return false
	}
}
