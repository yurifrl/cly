package builder

import (
	"fmt"
	"os"
	"path/filepath"

	blueprinttpl "github.com/NSXBet/nsx-cli/shared/blueprint/module"
	_ "github.com/NSXBet/nsx-cli/shared/blueprint/module/httpx"
	_ "github.com/NSXBet/nsx-cli/shared/blueprint/module/kafka"
	_ "github.com/NSXBet/nsx-cli/shared/blueprint/module/mysql"
	"github.com/NSXBet/nsx-cli/shared/interact"
)

// Run starts interactive wizard and generates project
func Run(args ProjectArguments) error {
	interact.Info("🎯 Welcome to NSX Blueprint Generator!")

	opts, err := askQuestions(args)
	if err != nil {
		return fmt.Errorf("failed to collect project information: %w", err)
	}

	destDir := filepath.Join("./", opts.ProjectName)
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("destination directory %s already exists", destDir)
	}

	if err := generateProject(destDir, opts); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	// Final step (Step 10): Update commit with all generated files
	interact.Info("📝 Updating commit with generated files...")
	if err := updateFinalCommit(destDir, opts.Debug); err != nil {
		interact.Error("❌ Failed to update commit!")
		return fmt.Errorf("failed to update commit: %w", err)
	}
	interact.Success("✅ Final commit updated!")

	interact.Success("🎉 Project %s generated successfully in %s", opts.ProjectName, destDir)
	interact.Info("💡 Next steps:")
	interact.Info("   1. cd %s", opts.ProjectName)
	interact.Info("   2. make run-all")
	interact.Info("   3. Start coding! 🚀")

	return nil
}

// generateProject orchestrates the entire project generation process
func generateProject(destDir string, opts *Options) error {
	interact.Info("🚀 Starting project generation...")

	// Step 1: Create base folder
	interact.Info("📦 Setting up base folder...")
	if err := createBaseFolder(destDir); err != nil {
		interact.Error("❌ Failed to setup base folder!")
		return fmt.Errorf("failed to setup base folder: %w", err)
	}
	interact.Success("✅ Base folder ready!")

	// Step 2: Initialize git repository and create initial commit (early, as some generation steps need git HEAD)
	interact.Info("🔧 Initializing Git repository...")
	if err := initGitRepo(destDir, opts.Debug); err != nil {
		interact.Error("❌ Failed to initialize Git repository!")
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}
	interact.Success("✅ Git repository initialized!")

	// Step 3: Generate all template files
	interact.Info("📝 Generating project files from templates...")
	if err := processEmbeddedDir(blueprinttpl.FS, "base", destDir, opts); err != nil {
		interact.Error("❌ Failed to process base templates!")
		return fmt.Errorf("failed to process base templates: %w", err)
	}
	for _, mod := range opts.SelectedModules {
		interact.Info("📁 Processing module: %s", mod.ID())
		if err := processEmbeddedDir(mod.TemplateFS(), mod.TemplateRoot(), destDir, opts); err != nil {
			interact.Error("❌ Failed to process module %s templates!", mod.ID())
			return fmt.Errorf("failed to process module %s templates: %w", mod.ID(), err)
		}
	}
	interact.Success("✅ All template files generated!")

	// Step 4: Create initial commit (so make gen can access HEAD)
	interact.Info("📝 Creating initial commit for code generation...")
	if err := createInitialCommit(destDir, opts.Debug); err != nil {
		interact.Error("❌ Failed to create initial commit!")
		return fmt.Errorf("failed to create initial commit: %w", err)
	}
	interact.Success("✅ Initial commit created!")

	// Step 5: Run code generation (protobuf, etc.)
	interact.Info("⚙️  Running code generation...")
	if err := runMakeGen(destDir, opts.Debug); err != nil {
		interact.Error("❌ Code generation failed!")
		return fmt.Errorf("code generation failed: %w", err)
	}
	interact.Success("✅ Code generation completed!")

	// Step 6: Fix dependency conflicts (containerd and protovalidate)
	interact.Info("🔧 Fixing dependency conflicts...")
	if err := fixDependencyConflicts(destDir, opts.Debug); err != nil {
		interact.Error("❌ Failed to fix dependency conflicts!")
		return fmt.Errorf("failed to fix dependency conflicts: %w", err)
	}
	interact.Success("✅ Dependency conflicts fixed!")

	// Step 7: Run go mod tidy to organize dependencies
	interact.Info("🔧 Organizing Go dependencies...")
	if err := runGoModTidy(destDir, opts.Debug); err != nil {
		interact.Error("❌ Go mod tidy failed!")
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	interact.Success("✅ Go dependencies organized!")

	// Step 8: Run make format to format code
	interact.Info("🔧 Formatting code...")
	if err := runMakeFormat(destDir, opts.Debug); err != nil {
		interact.Error("❌ Failed to format code!")
		return fmt.Errorf("failed to format code: %w", err)
	}
	interact.Success("✅ Code formatted!")

	// Step 9: Fetch and create golangci-lint configuration
	interact.Info("📋 Setting up golangci-lint configuration...")
	if err := FetchGolangCIConfig(opts.ProxyURL, destDir, opts.Debug); err != nil {
		interact.Error("❌ Failed to setup golangci-lint configuration!")
		return fmt.Errorf("failed to setup golangci-lint configuration: %w", err)
	}
	interact.Success("✅ Golangci-lint configuration created!")

	// Step 10: Install wiz-cursor https://github.com/NSXBet/wiz-cursor
	interact.Info("🧙 Installing the wiz-cursor -- https://github.com/NSXBet/wiz-cursor --...")
	if err := installWizCursor(destDir, opts.Debug); err != nil {
		interact.Error("❌ Failed to install wiz-cursor!")
		return fmt.Errorf("failed to install wiz-cursor: %w", err)
	}
	interact.Success("✅ Wiz-cursor installed successfully!")

	return nil
}
