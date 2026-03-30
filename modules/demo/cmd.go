package demo

import (
	altscreentoggle "github.com/yurifrl/cly/modules/demo/altscreen-toggle"
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/demo/autocomplete"
	"github.com/yurifrl/cly/modules/demo/canvas"
	"github.com/yurifrl/cly/modules/demo/capability"
	"github.com/yurifrl/cly/modules/demo/cellbuffer"
	"github.com/yurifrl/cly/modules/demo/chat"
	"github.com/yurifrl/cly/modules/demo/clickable"
	colorprofilemod "github.com/yurifrl/cly/modules/demo/colorprofile"
	composableviews "github.com/yurifrl/cly/modules/demo/composable-views"
	cursorstyle "github.com/yurifrl/cly/modules/demo/cursor-style"
	lgblend1d "github.com/yurifrl/cly/modules/demo/lg-blend-1d"
	lgblend2d "github.com/yurifrl/cly/modules/demo/lg-blend-2d"
	lgblendrotation "github.com/yurifrl/cly/modules/demo/lg-blend-rotation"
	lgbrightness "github.com/yurifrl/cly/modules/demo/lg-brightness"
	lgcolor "github.com/yurifrl/cly/modules/demo/lg-color"
	lgcolordialog "github.com/yurifrl/cly/modules/demo/lg-color-dialog"
	lgcompat "github.com/yurifrl/cly/modules/demo/lg-compat"
	lglayout "github.com/yurifrl/cly/modules/demo/lg-layout"
	lglistduckduckgoose "github.com/yurifrl/cly/modules/demo/lg-list-duckduckgoose"
	lglistglow "github.com/yurifrl/cly/modules/demo/lg-list-glow"
	lglistgrocery "github.com/yurifrl/cly/modules/demo/lg-list-grocery"
	lglistroman "github.com/yurifrl/cly/modules/demo/lg-list-roman"
	lgtableansi "github.com/yurifrl/cly/modules/demo/lg-table-ansi"
	lgtablechess "github.com/yurifrl/cly/modules/demo/lg-table-chess"
	lgtablelanguages "github.com/yurifrl/cly/modules/demo/lg-table-languages"
	lgtablemindy "github.com/yurifrl/cly/modules/demo/lg-table-mindy"
	lgtablepokemon "github.com/yurifrl/cly/modules/demo/lg-table-pokemon"
	lgtreebackground "github.com/yurifrl/cly/modules/demo/lg-tree-background"
	lgtreefiles "github.com/yurifrl/cly/modules/demo/lg-tree-files"
	lgtreemakeup "github.com/yurifrl/cly/modules/demo/lg-tree-makeup"
	lgtreerounded "github.com/yurifrl/cly/modules/demo/lg-tree-rounded"
	lgtreesimple "github.com/yurifrl/cly/modules/demo/lg-tree-simple"
	lgtreestyles "github.com/yurifrl/cly/modules/demo/lg-tree-styles"
	lgtreetoggle "github.com/yurifrl/cly/modules/demo/lg-tree-toggle"
	creditcardform "github.com/yurifrl/cly/modules/demo/credit-card-form"
	"github.com/yurifrl/cly/modules/demo/debounce"
	doomfire "github.com/yurifrl/cly/modules/demo/doom-fire"
	dynamictextarea "github.com/yurifrl/cly/modules/demo/dynamic-textarea"
	"github.com/yurifrl/cly/modules/demo/exec"
	"github.com/yurifrl/cly/modules/demo/eyes"
	filepicker "github.com/yurifrl/cly/modules/demo/file-picker"
	focusblur "github.com/yurifrl/cly/modules/demo/focus-blur"
	"github.com/yurifrl/cly/modules/demo/fullscreen"
	"github.com/yurifrl/cly/modules/demo/glamour"
	"github.com/yurifrl/cly/modules/demo/help"
	"github.com/yurifrl/cly/modules/demo/http"
	isbnform "github.com/yurifrl/cly/modules/demo/isbn-form"
	keyboardenhancements "github.com/yurifrl/cly/modules/demo/keyboard-enhancements"
	listdefault "github.com/yurifrl/cly/modules/demo/list-default"
	listfancy "github.com/yurifrl/cly/modules/demo/list-fancy"
	listsimple "github.com/yurifrl/cly/modules/demo/list-simple"
	lglistsimple "github.com/yurifrl/cly/modules/demo/lg-list-simple"
	"github.com/yurifrl/cly/modules/demo/mouse"
	"github.com/yurifrl/cly/modules/demo/pager"
	"github.com/yurifrl/cly/modules/demo/paginator"
	"github.com/yurifrl/cly/modules/demo/pipe"
	preventquit "github.com/yurifrl/cly/modules/demo/prevent-quit"
	printkey "github.com/yurifrl/cly/modules/demo/print-key"
	progressanimated "github.com/yurifrl/cly/modules/demo/progress-animated"
	progressdownload "github.com/yurifrl/cly/modules/demo/progress-download"
	progressbar "github.com/yurifrl/cly/modules/demo/progress-bar"
	progressstatic "github.com/yurifrl/cly/modules/demo/progress-static"
	queryterm "github.com/yurifrl/cly/modules/demo/query-term"
	"github.com/yurifrl/cly/modules/demo/realtime"
	"github.com/yurifrl/cly/modules/demo/result"
	sendmsg "github.com/yurifrl/cly/modules/demo/send-msg"
	"github.com/yurifrl/cly/modules/demo/sequence"
	setterminalcolor "github.com/yurifrl/cly/modules/demo/set-terminal-color"
	setwindowtitle "github.com/yurifrl/cly/modules/demo/set-window-title"
	"github.com/yurifrl/cly/modules/demo/simple"
	"github.com/yurifrl/cly/modules/demo/space"
	"github.com/yurifrl/cly/modules/demo/splash"
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
	"github.com/yurifrl/cly/modules/demo/vanish"
	"github.com/yurifrl/cly/modules/demo/views"
	windowsize "github.com/yurifrl/cly/modules/demo/window-size"
)

var DemoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Component demonstrations",
	Long:  "Interactive demonstrations of Bubbletea, Bubbles, Huh, and Lipgloss components",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOverlay(cmd)
	},
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
	lgcolordialog.Register(DemoCmd)
	lglayout.Register(DemoCmd)
	lglistsimple.Register(DemoCmd)
	lglistgrocery.Register(DemoCmd)
	lglistroman.Register(DemoCmd)
	lglistduckduckgoose.Register(DemoCmd)
	lglistglow.Register(DemoCmd)
	lgtreesimple.Register(DemoCmd)
	lgtreefiles.Register(DemoCmd)
	lgtreemakeup.Register(DemoCmd)
	lgtreerounded.Register(DemoCmd)
	lgtreestyles.Register(DemoCmd)
	lgtreetoggle.Register(DemoCmd)
	lgtreebackground.Register(DemoCmd)
	lgtableansi.Register(DemoCmd)
	lgtablechess.Register(DemoCmd)
	lgtablelanguages.Register(DemoCmd)
	lgtablepokemon.Register(DemoCmd)
	lgtablemindy.Register(DemoCmd)
	lgblend1d.Register(DemoCmd)
	lgblend2d.Register(DemoCmd)
	lgblendrotation.Register(DemoCmd)
	lgbrightness.Register(DemoCmd)
	lgcolor.Register(DemoCmd)
	lgcompat.Register(DemoCmd)
	canvas.Register(DemoCmd)
	capability.Register(DemoCmd)
	clickable.Register(DemoCmd)
	colorprofilemod.Register(DemoCmd)
	cursorstyle.Register(DemoCmd)
	doomfire.Register(DemoCmd)
	dynamictextarea.Register(DemoCmd)
	isbnform.Register(DemoCmd)
	keyboardenhancements.Register(DemoCmd)
	printkey.Register(DemoCmd)
	progressbar.Register(DemoCmd)
	queryterm.Register(DemoCmd)
	setterminalcolor.Register(DemoCmd)
	space.Register(DemoCmd)
	splash.Register(DemoCmd)
	vanish.Register(DemoCmd)
}

func Register(parent *cobra.Command) {
	parent.AddCommand(DemoCmd)
}
