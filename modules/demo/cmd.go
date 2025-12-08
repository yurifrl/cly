package demo

import (
	altscreentoggle "github.com/yurifrl/cly/modules/demo/altscreen-toggle"
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/demo/autocomplete"
	"github.com/yurifrl/cly/modules/demo/cellbuffer"
	"github.com/yurifrl/cly/modules/demo/chat"
	composableviews "github.com/yurifrl/cly/modules/demo/composable-views"
	creditcardform "github.com/yurifrl/cly/modules/demo/credit-card-form"
	"github.com/yurifrl/cly/modules/demo/debounce"
	"github.com/yurifrl/cly/modules/demo/exec"
	"github.com/yurifrl/cly/modules/demo/eyes"
	filepicker "github.com/yurifrl/cly/modules/demo/file-picker"
	focusblur "github.com/yurifrl/cly/modules/demo/focus-blur"
	"github.com/yurifrl/cly/modules/demo/fullscreen"
	"github.com/yurifrl/cly/modules/demo/glamour"
	"github.com/yurifrl/cly/modules/demo/help"
	"github.com/yurifrl/cly/modules/demo/http"
	listdefault "github.com/yurifrl/cly/modules/demo/list-default"
	listfancy "github.com/yurifrl/cly/modules/demo/list-fancy"
	listsimple "github.com/yurifrl/cly/modules/demo/list-simple"
	"github.com/yurifrl/cly/modules/demo/mouse"
	"github.com/yurifrl/cly/modules/demo/pager"
	"github.com/yurifrl/cly/modules/demo/paginator"
	"github.com/yurifrl/cly/modules/demo/pipe"
	preventquit "github.com/yurifrl/cly/modules/demo/prevent-quit"
	progressanimated "github.com/yurifrl/cly/modules/demo/progress-animated"
	progressdownload "github.com/yurifrl/cly/modules/demo/progress-download"
	progressstatic "github.com/yurifrl/cly/modules/demo/progress-static"
	"github.com/yurifrl/cly/modules/demo/realtime"
	"github.com/yurifrl/cly/modules/demo/result"
	sendmsg "github.com/yurifrl/cly/modules/demo/send-msg"
	"github.com/yurifrl/cly/modules/demo/sequence"
	setwindowtitle "github.com/yurifrl/cly/modules/demo/set-window-title"
	"github.com/yurifrl/cly/modules/demo/simple"
	packagemanager "github.com/yurifrl/cly/modules/demo/package-manager"
	spliteditors "github.com/yurifrl/cly/modules/demo/split-editors"
	"github.com/yurifrl/cly/modules/demo/spinner"
	"github.com/yurifrl/cly/modules/demo/spinners"
	"github.com/yurifrl/cly/modules/demo/stopwatch"
	"github.com/yurifrl/cly/modules/demo/suspend"
	"github.com/yurifrl/cly/modules/demo/table"
	tableresize "github.com/yurifrl/cly/modules/demo/table-resize"
	"github.com/yurifrl/cly/modules/demo/tabs"
	"github.com/yurifrl/cly/modules/demo/textarea"
	"github.com/yurifrl/cly/modules/demo/textinput"
	"github.com/yurifrl/cly/modules/demo/textinputs"
	"github.com/yurifrl/cly/modules/demo/timer"
	tuidaemoncombo "github.com/yurifrl/cly/modules/demo/tui-daemon-combo"
	"github.com/yurifrl/cly/modules/demo/views"
	windowsize "github.com/yurifrl/cly/modules/demo/window-size"
)

var DemoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Component demonstrations",
	Long:  "Interactive demonstrations of Bubbletea, Bubbles, Huh, and Lipgloss components",
}

func init() {
	chat.Register(DemoCmd)
	debounce.Register(DemoCmd)
	exec.Register(DemoCmd)
	eyes.Register(DemoCmd)
	focusblur.Register(DemoCmd)
	mouse.Register(DemoCmd)
	pipe.Register(DemoCmd)
	preventquit.Register(DemoCmd)
	suspend.Register(DemoCmd)
	tuidaemoncombo.Register(DemoCmd)
	tableresize.Register(DemoCmd)
	spinner.Register(DemoCmd)
	table.Register(DemoCmd)
	listsimple.Register(DemoCmd)
	textinput.Register(DemoCmd)
	textarea.Register(DemoCmd)
	progressstatic.Register(DemoCmd)
	progressanimated.Register(DemoCmd)
	progressdownload.Register(DemoCmd)
	stopwatch.Register(DemoCmd)
	timer.Register(DemoCmd)
	autocomplete.Register(DemoCmd)
	cellbuffer.Register(DemoCmd)
	creditcardform.Register(DemoCmd)
	filepicker.Register(DemoCmd)
	listdefault.Register(DemoCmd)
	listfancy.Register(DemoCmd)
	help.Register(DemoCmd)
	simple.Register(DemoCmd)
	spinners.Register(DemoCmd)
	tabs.Register(DemoCmd)
	textinputs.Register(DemoCmd)
	windowsize.Register(DemoCmd)
	realtime.Register(DemoCmd)
	result.Register(DemoCmd)
	sendmsg.Register(DemoCmd)
	sequence.Register(DemoCmd)
	setwindowtitle.Register(DemoCmd)
	altscreentoggle.Register(DemoCmd)
	fullscreen.Register(DemoCmd)
	glamour.Register(DemoCmd)
	http.Register(DemoCmd)
	packagemanager.Register(DemoCmd)
	pager.Register(DemoCmd)
	paginator.Register(DemoCmd)
	views.Register(DemoCmd)
	composableviews.Register(DemoCmd)
	spliteditors.Register(DemoCmd)
}

func Register(parent *cobra.Command) {
	parent.AddCommand(DemoCmd)
}
