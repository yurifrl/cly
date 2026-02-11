package claudetasks

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "claude-tasks [name]",
		Aliases: []string{"ct"},
		Short:   "Manage Claude Code task lists",
		Args:    cobra.MaximumNArgs(1),
		RunE:    createRun,
	}

	cmd.AddCommand(listCmd())
	cmd.AddCommand(deleteCmd())

	parent.AddCommand(cmd)
}

func createRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	name := args[0]
	filePath := FilePath()

	store, err := Load(filePath)
	if err != nil {
		return err
	}

	store[name] = TaskList{Name: name}

	if err := Save(filePath, store); err != nil {
		return err
	}

	fmt.Printf("Created task list %q\n", name)
	return nil
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List task lists",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := Load(FilePath())
			if err != nil {
				return err
			}

			if len(store) == 0 {
				fmt.Println("No task lists")
				return nil
			}

			for _, tl := range store {
				fmt.Println(tl.Name)
			}
			return nil
		},
	}
}

func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a task list",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			store, err := Load(FilePath())
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var names []string
			for _, tl := range store {
				names = append(names, tl.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			filePath := FilePath()

			store, err := Load(filePath)
			if err != nil {
				return err
			}

			if _, ok := store[name]; !ok {
				return fmt.Errorf("task list %q not found", name)
			}

			delete(store, name)

			if err := Save(filePath, store); err != nil {
				return err
			}

			fmt.Printf("Deleted task list %q\n", name)
			return nil
		},
	}
}
