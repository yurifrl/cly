package builder

import (
	"fmt"
	"slices"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/samber/lo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/NSXBet/nsx-cli/shared/blueprint/module"
	"github.com/NSXBet/nsx-cli/shared/interact"
)

// askQuestions collects project information from user input
func askQuestions(args ProjectArguments) (*Options, error) {
	var qs []*survey.Question

	// Only ask for project name if not provided via flags
	if args.ProjectName == "" {
		qs = append(qs, &survey.Question{
			Name:     "projectName",
			Prompt:   &survey.Input{Message: "Project name:"},
			Validate: survey.Required,
		})
	}

	// Only ask for team if not provided via flags
	if args.Team == "" {
		qs = append(qs, &survey.Question{
			Name:     "team",
			Prompt:   &survey.Input{Message: "Team:"},
			Validate: survey.Required,
		})
	}

	answers := struct {
		ProjectName string
		Team        string
	}{
		// Pre-fill with provided arguments
		ProjectName: args.ProjectName,
		Team:        args.Team,
	}

	// Only ask questions if there are any to ask
	if len(qs) > 0 {
		if err := survey.Ask(qs, &answers); err != nil {
			return nil, err
		}
	}

	projectName := sanitizeProjectName(answers.ProjectName)
	opts := &Options{
		ProjectName:           projectName,
		ProjectNamePascalCase: ToPascalCase(projectName),
		ProjectNameSnakeCase:  ToSnakeCase(projectName),
		ProjectNameTitle:      cases.Title(language.English).String(projectName),
		Team:                  answers.Team,
		Debug:                 args.Debug,
		ProxyURL:              args.ProxyURL,
	}

	// modules selection
	if len(module.AllModules) > 0 {
		// Group modules by type
		modulesByType := make(map[string][]module.Module)
		interact.Info(
			"📦 Organizing available modules... %v",
			lo.Map(module.AllModules, func(m module.Module, _ int) string { return m.ID() }),
		)
		for _, m := range module.AllModules {
			moduleType := string(m.Type()) // Convert ModuleType to string
			modulesByType[moduleType] = append(modulesByType[moduleType], m)
		}

		// Create organized options for display
		var moduleOptions []string
		var moduleMapping []module.Module // To map back from selection to module

		// Sort categories for consistent display
		categoryOrder := []string{"Database", "Handler", "Caching", "Messaging", "Observability", "Security"}

		for _, categoryName := range categoryOrder {
			if modules, exists := modulesByType[categoryName]; exists {
				for _, m := range modules {
					option := fmt.Sprintf("[%s] %s - %s", categoryName, m.ID(), m.Description())
					moduleOptions = append(moduleOptions, option)
					moduleMapping = append(moduleMapping, m)
				}
			}
		}

		// Add any remaining categories not in the predefined order
		for categoryName, modules := range modulesByType {
			if !slices.Contains(categoryOrder, categoryName) {
				for _, m := range modules {
					option := fmt.Sprintf("[%s] %s - %s", categoryName, m.ID(), m.Description())
					moduleOptions = append(moduleOptions, option)
					moduleMapping = append(moduleMapping, m)
				}
			}
		}

		var selected []string
		prompt := &survey.MultiSelect{
			Message: "Select additional modules (space to select):",
			Options: moduleOptions,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			return nil, err
		}

		// Process selections by finding corresponding modules
		for _, selectedOption := range selected {
			for i, option := range moduleOptions {
				if option == selectedOption && i < len(moduleMapping) {
					mod := moduleMapping[i]
					opts.SelectedModules = append(opts.SelectedModules, mod)
					if modWithImports, ok := mod.(module.ModuleWithImports); ok {
						if opts.Imports == nil {
							opts.Imports = make(map[string]module.ImportInfo)
						}
						importInfo := modWithImports.Imports(opts.ProjectName)
						for cmdFileName, imports := range importInfo {
							if existing, exists := opts.Imports[cmdFileName]; exists {
								// Merge existing imports with new ones
								existing.Paths = append(existing.Paths, imports.Paths...)
								existing.FxImports = slices.Compact(append(existing.FxImports, imports.FxImports...))
								opts.Imports[cmdFileName] = existing
							} else {
								opts.Imports[cmdFileName] = imports
							}
						}
					}
					break
				}
			}
		}
	}

	return opts, nil
}

// sanitizeProjectName cleans and formats project name
func sanitizeProjectName(name string) string {
	cleaned := strings.TrimSpace(name)
	cleaned = strings.ToLower(cleaned)
	return strings.ReplaceAll(cleaned, " ", "-")
}
