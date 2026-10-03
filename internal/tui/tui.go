package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"

	"voice-snooter/internal/config"
	"voice-snooter/internal/model"
)

type screen struct {
	state                      State
	states                     <-chan State
	actions                    chan<- Action
	ctx                        context.Context
	cancel                     context.CancelFunc
	configPath                 string
	width, height, tab, offset int
	pending                    string
	selected                   int
	picker                     *sourcePicker
	now                        time.Time
}
type stateMsg State
type endedMsg struct{}
type tickMsg time.Time

func noticeTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (s screen) next() tea.Cmd {
	return func() tea.Msg {
		select {
		case v, ok := <-s.states:
			if !ok {
				return endedMsg{}
			}
			return stateMsg(v)
		case <-s.ctx.Done():
			return endedMsg{}
		}
	}
}
func (s screen) Init() tea.Cmd { return tea.Batch(s.next(), noticeTick()) }
func (s screen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case stateMsg:
		key := ""
		if rows := s.rules(); len(rows) > 0 {
			key = rows[min(s.selected, len(rows)-1)].key
		}
		s.state = State(m)
		s.pending = ""
		for n, row := range s.rules() {
			if row.key == key {
				s.selected = n
			}
		}
		s.selected = min(s.selected, max(0, len(s.rules())-1))
		if s.picker != nil && s.picker.revision != s.state.Revision {
			s.picker = nil
			s.pending = "Choices changed; reopen Source"
		}
		return s, s.next()
	case tickMsg:
		s.now = time.Time(m)
		return s, noticeTick()
	case endedMsg:
		return s, tea.Quit
	case tea.WindowSizeMsg:
		s.width = m.Width
		s.height = m.Height
	case tea.KeyPressMsg:
		if s.picker != nil && m.String() != "q" && m.String() != "ctrl+c" {
			s.pickerKey(m.String())
			return s, nil
		}
		switch m.String() {
		case "q", "ctrl+c":
			s.cancel()
			return s, tea.Quit
		case "tab", "right":
			s.tab = (s.tab + 1) % 4
			s.offset = 0
		case "shift+tab", "left":
			s.tab = (s.tab + 3) % 4
			s.offset = 0
		case "down", "j":
			if s.tab == 0 {
				s.selected = min(s.selected+1, max(0, len(s.rules())-1))
				s.offset = max(0, s.selected+4-max(1, s.height-6)+1)
			} else {
				s.offset++
			}
		case "up", "k":
			if s.tab == 0 {
				s.selected = max(0, s.selected-1)
				s.offset = min(s.offset, s.selected+3)
			} else {
				s.offset = max(0, s.offset-1)
			}
		case "home":
			s.offset = 0
		case "pgdown":
			s.offset += max(1, s.height-8)
		case "pgup":
			s.offset = max(0, s.offset-max(1, s.height-8))
		case "enter":
			if s.tab == 0 {
				s.ruleAction("enter")
			}
		case "x":
			if s.tab == 0 {
				s.queue(Action{Kind: resetChoices, Revision: s.state.Revision})
			}
		case "l", "r", "f", " ", "space":
			if (m.String() == " " || m.String() == "space") && s.tab == 0 {
				s.ruleAction(" ")
				break
			}
			a := Refresh
			if m.String() == "l" {
				a = ToggleLive
			}
			if m.String() == "r" {
				a = Reload
			}
			select {
			case s.actions <- a:
				s.pending = "Queued"
			default:
				s.pending = "A command is already queued"
			}
		}
	}
	return s, nil
}

// Device names and errors are untrusted terminal text: strip escape/control
// characters so a device label cannot inject terminal commands or extra rows.
func clean(v string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, ansi.Strip(v))
}
func (s screen) lines() []string {
	rows := []string{}
	switch s.tab {
	case 0:
		return s.ruleLines()
	case 1:
		rows = append(rows, "ROUTING RULES / DESIRED STATE")
		if s.state.Plan == nil {
			rows = append(rows, "Waiting for a valid routing plan...")
		} else if t := s.state.Plan.Topology; t != nil {
			rows = append(rows, fmt.Sprintf("Volt ASIO: %t    Playback output: %s", t.ASIOActive, t.PlaybackTarget))
			for _, op := range t.Operations {
				mark := " = "
				if op.Change {
					mark = " > "
				}
				if op.Device != nil {
					rows = append(rows, fmt.Sprintf("%s%-10s %s [%s]", mark, op.Target, empty(op.Device.Name), op.Device.Driver))
				} else {
					rows = append(rows, fmt.Sprintf("%s%-20s %g -> %d", mark, op.Parameter, op.BeforeValue, op.Value))
				}
			}
			for _, reason := range t.Unresolved {
				rows = append(rows, "UNRESOLVED: "+reason)
			}
		} else {
			for _, d := range s.state.Plan.Decisions {
				name := "unresolved"
				if d.Desired != nil {
					name = d.Desired.Name
				}
				rows = append(rows, fmt.Sprintf("%s: %s -> %s (change=%t)", d.Target, empty(d.Current), name, d.Change))
			}
		}
		rows = append(rows, "", "CURRENT ASSIGNMENTS")
		for _, target := range model.Slots(s.state.Snapshot.Edition) {
			rows = append(rows, fmt.Sprintf("%-10s %s", target, empty(s.state.Snapshot.Assignments[target])))
		}
	case 2:
		rows = append(rows, "DEVICE INVENTORY  (WDM presence; ASIO entries alone do not prove connection)")
		for _, d := range s.state.Snapshot.Devices {
			rows = append(rows, fmt.Sprintf("%-6s %-5s %s", d.Direction, d.Driver, d.Name))
		}
		if len(s.state.Snapshot.Devices) == 0 {
			rows = append(rows, "No inventory yet.")
		}
	case 3:
		rows = append(rows, "RECENT EVENTS  (newest first, last 50)")
		for i := len(s.state.Events) - 1; i >= 0; i-- {
			rows = append(rows, s.state.Events[i])
		}
	}
	return rows
}
func empty(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
func (s screen) View() tea.View { return s.dashboardView() }

func Run(ctx context.Context, cfg config.Config, path, dll string, live bool, out io.Writer, deps Dependencies) error {
	file, ok := out.(*os.File)
	if !ok || !term.IsTerminal(file.Fd()) || !term.IsTerminal(os.Stdin.Fd()) {
		return fmt.Errorf("tui requires an interactive terminal; use watch or plan for redirected output")
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	actions := make(chan Action, 8)
	states := make(chan State, 1)
	done := make(chan struct{})
	go work(runCtx, cfg, path, dll, live, deps, actions, states, done)
	initial := screen{ctx: runCtx, cancel: cancel, states: states, actions: actions, configPath: path, state: State{Intent: cfg.VoiceIntent(), StateError: cfg.StateError}}
	// Windows VT terminals often omit TERM; use our 256-color palette explicitly.
	profile := colorprofile.ANSI256
	if os.Getenv("NO_COLOR") != "" {
		profile = colorprofile.Ascii
	}
	final, err := tea.NewProgram(initial, tea.WithColorProfile(profile), tea.WithContext(runCtx), tea.WithOutput(out), tea.WithInput(os.Stdin)).Run()
	cancel()
	<-done
	if view, ok := final.(screen); ok && view.state.Intent != nil && view.state.Intent.Recording != nil {
		fmt.Fprintln(out, "Recorder transport was left unchanged. Any active recording continues in Voicemeeter.")
	}
	if errors.Is(err, tea.ErrProgramKilled) && runCtx.Err() != nil {
		return nil
	}
	return err
}
