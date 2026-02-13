package agents

import (
	"os"
	"path/filepath"
	"strings"
)

// SyncItem represents a single file to sync from source to target.
type SyncItem struct {
	Source    string
	Target    string
	Transform TransformKind
}

// SyncPlan is the list of items to sync for one IDE.
type SyncPlan struct {
	Items []SyncItem
}

// Discover walks the source .agents directory and produces a SyncPlan for the given IDE.
func Discover(sourceDir string, ide *IDEDef, targetBase string) (*SyncPlan, error) {
	var plan SyncPlan

	// Phase 1: Shared configs (root of .agents)
	if err := discoverShared(sourceDir, ide, targetBase, &plan); err != nil {
		return nil, err
	}

	// Phase 2: IDE-specific overrides from .agents/ides/<ide>/
	ideDir := filepath.Join(sourceDir, "ides", ide.Name)
	if err := discoverIDESpecific(ideDir, ide, targetBase, &plan); err != nil {
		return nil, err
	}

	return &plan, nil
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

			rel, _ := filepath.Rel(srcSub, path)
			targetPath := filepath.Join(targetBase, targetSubdir, rel)
			transform := classifyTransform(path, ide)

			// JSONC files get .json extension in target
			if strings.HasSuffix(targetPath, ".jsonc") && transform == TransformJSONCSK {
				targetPath = strings.TrimSuffix(targetPath, ".jsonc") + ".json"
			}

			plan.Items = append(plan.Items, SyncItem{
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
		plan.Items = append(plan.Items, SyncItem{
			Source:    agentsMD,
			Target:    filepath.Join(targetBase, ide.AgentsMDTarget),
			Transform: TransformNone,
		})
	}

	// Special files (e.g. claude.json → settings.json)
	for srcName, dstName := range ide.SpecialFiles {
		srcPath := filepath.Join(sourceDir, srcName)
		if _, err := os.Stat(srcPath); err == nil {
			transform := TransformNone
			if strings.HasSuffix(srcName, ".jsonc") {
				transform = TransformJSONCSK
			}
			plan.Items = append(plan.Items, SyncItem{
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

		rel, _ := filepath.Rel(ideDir, path)
		targetPath := filepath.Join(targetBase, rel)
		transform := classifyTransform(path, ide)

		if strings.HasSuffix(targetPath, ".jsonc") && transform == TransformJSONCSK {
			targetPath = strings.TrimSuffix(targetPath, ".jsonc") + ".json"
		}

		plan.Items = append(plan.Items, SyncItem{
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
