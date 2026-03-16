package dotfiles

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/style"
)

func registerJobsCommands(parent *cobra.Command) {
	jobsCmd := &cobra.Command{
		Use:   "jobs",
		Short: "Manage declarative startup and scheduled jobs",
	}

	jobsCmd.AddCommand(
		&cobra.Command{
			Use:   "apply",
			Short: "Apply jobs declared in dotfiles.conf",
			RunE:  runJobsApply,
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show status of declared jobs",
			RunE:  runJobsStatus,
		},
		&cobra.Command{
			Use:   "remove",
			Short: "Remove jobs declared in dotfiles.conf",
			RunE:  runJobsRemove,
		},
	)

	parent.AddCommand(jobsCmd)
}

func runJobsApply(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadDotfilesConfig()
	if err != nil {
		return err
	}
	if len(cfg.Jobs) == 0 {
		fmt.Println("No jobs declared.")
		return nil
	}
	for _, e := range cfg.Errors {
		fmt.Printf("⚠️  %s\n", e)
	}
	if err := ApplyJobs(cfg, JobApplyOptions{Force: forceFlag}); err != nil {
		return err
	}
	fmt.Printf("%s Applied %d job(s)\n", style.GreenStyle.Render("✅"), len(cfg.Jobs))
	return nil
}

func runJobsStatus(cmd *cobra.Command, args []string) error {
	cfg, configPath, err := loadDotfilesConfig()
	if err != nil {
		return err
	}
	if len(cfg.Jobs) == 0 {
		fmt.Println("No jobs declared.")
		return nil
	}
	statuses, err := StatusJobs(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("Jobs: %s\n\n", configPath)
	for _, status := range statuses {
		switch status.Run {
		case JobRunOnce:
			icon := style.SubtleStyle.Render("○")
			state := "pending"
			if status.Completed {
				icon = style.GreenStyle.Render("✓")
				state = "done"
			}
			fmt.Printf("%s %-18s %-8s %s\n", icon, status.Name, status.Run, state)
		default:
			icon := style.SubtleStyle.Render("○")
			state := "not registered"
			if status.Registered {
				icon = style.GreenStyle.Render("✓")
				state = "registered"
			}
			extra := ""
			if status.Run == JobRunInterval {
				extra = " every=" + status.Every
			}
			if status.KeepAlive {
				extra += " keepalive"
			}
			fmt.Printf("%s %-18s %-8s %s%s\n", icon, status.Name, status.Run, state, extra)
		}
	}
	return nil
}

func runJobsRemove(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadDotfilesConfig()
	if err != nil {
		return err
	}
	if len(cfg.Jobs) == 0 {
		fmt.Println("No jobs declared.")
		return nil
	}
	if err := RemoveJobs(cfg); err != nil {
		return err
	}
	fmt.Printf("%s Removed %d job(s)\n", style.GreenStyle.Render("✅"), len(cfg.Jobs))
	return nil
}

func loadDotfilesConfig() (*Config, string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("config not found: %s", configPath)
	}
	cfg, err := ParseConfig(configPath)
	if err != nil {
		return nil, "", err
	}
	return cfg, configPath, nil
}
