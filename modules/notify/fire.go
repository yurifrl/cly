package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/notify"
	"github.com/yurifrl/cly/pkg/style"
)

func createFireCmd() *cobra.Command {
	var title, message, sound, group string

	cmd := &cobra.Command{
		Use:   "fire",
		Short: "Fire a single native macOS notification",
		Long: `Send one notification through the native macOS daemon.

Example:
  cly notify fire --title "Backup done" --message "Wrote 1.2GB"
  cly notify fire --title "Alert" --message "..." --sound Sosumi`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			native := notify.NewNativeMacOSNotifier(ctx)
			if !native.Available() {
				return fmt.Errorf("native macOS notifier unavailable; run `cly u` to install signed bundle")
			}

			if group == "" {
				group = "cly.fire"
			}

			err := native.Send(ctx, notify.Notification{
				Title:   title,
				Message: message,
				Sound:   sound,
				Group:   group,
			})
			if err != nil {
				return err
			}

			// UN.add() is async — wait briefly so daemon delivers before we exit.
			time.Sleep(750 * time.Millisecond)
			fmt.Println(style.GreenStyle.Render("✓ fired"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "cly", "notification title")
	cmd.Flags().StringVarP(&message, "message", "m", "", "notification body")
	cmd.Flags().StringVar(&sound, "sound", "Glass", "macOS sound name (Glass, Basso, Sosumi, Ping, ...)")
	cmd.Flags().StringVar(&group, "group", "", "notification group; same group replaces previous")
	return cmd
}
