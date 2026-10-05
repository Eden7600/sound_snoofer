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

	"sound-snoofer/internal/config"
)

type screen struct {
	inflightAction             Action
	draft                      *config.Intent
	deferredEdits              []settingEdit
	inflight                   uint64
	nextEditID                 uint64
	attached                   bool
	state                      State
	states                     <-chan State
	actions                    chan<- Action
	ctx                        context.Context
	cancel                     context.CancelFunc
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
		if s.picker != nil {
			s.refreshPicker(s.picker.options[s.picker.selected])
		}
		s.pending = ""
		s.acceptEditState()
		for n, row := range s.rules() {
			if row.key == key {
				s.selected = n
			}
		}
		s.selected = min(s.selected, max(0, len(s.rules())-1))
		if s.picker != nil && s.picker.revision != s.state.Revision && s.inflight == 0 {
			s.picker = nil
			s.pending = "Choices changed; reopen Source"
		}
		return s, s.next()
	case tickMsg:
		s.now = time.Time(m)
		if !s.state.ObservedAt.IsZero() && s.now.Sub(s.state.ObservedAt) > 5*time.Second {
			s.state.Health = "Audio worker stalled; native recovery unavailable"
		}
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
			s.tab = (s.tab + 1) % 2
			s.offset = 0
		case "shift+tab", "left":
			s.tab = (s.tab + 1) % 2
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
			s.queue(a)
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
	actions, states, done := StartWorker(runCtx, cfg, path, dll, live, deps)
	err := RunConnected(runCtx, State{Intent: cfg.VoiceIntent(), StateError: cfg.StateError}, actions, states, os.Stdin, out, false)
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		return fmt.Errorf("audio worker did not stop within 3 seconds")
	}
	return err
}

// StartWorker starts the sole audio actor. The caller owns context cancellation;
// the worker closes states and done. Consumers must treat published state as immutable.
func StartWorker(ctx context.Context, cfg config.Config, path, dll string, live bool, deps Dependencies) (chan<- Action, <-chan State, <-chan struct{}) {
	actions := make(chan Action, 8)
	states := make(chan State, 1)
	done := make(chan struct{})
	go work(ctx, cfg, path, dll, live, deps, actions, states, done)
	return actions, states, done
}

// RunConnected displays controls over an existing actor connection. Closing the
// view cancels only its readers, never the actor supplied by the caller.
func RunConnected(ctx context.Context, state State, actions chan<- Action, states <-chan State, in io.Reader, out io.Writer, attached bool) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	initial := screen{ctx: runCtx, cancel: cancel, states: states, actions: actions, state: state, nextEditID: state.EditAck, attached: attached}
	if attached {
		initial.nextEditID = max(state.EditAck, uint64(time.Now().UnixNano()))
	}
	// Windows VT terminals often omit TERM; use our 256-color palette explicitly.
	profile := colorprofile.ANSI256
	if os.Getenv("NO_COLOR") != "" {
		profile = colorprofile.Ascii
	}
	_, err := tea.NewProgram(initial, tea.WithColorProfile(profile), tea.WithContext(runCtx), tea.WithOutput(out), tea.WithInput(in)).Run()
	if errors.Is(err, tea.ErrProgramKilled) && runCtx.Err() != nil {
		return nil
	}
	return err
}
