// Package app provides Snoofer's domain-independent desktop shell.
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"sound-snoofer/snoofer"
)

// ViewState crosses the private controls pipe without domain-specific types.
type ViewState struct {
	Controls     []snoofer.Control
	Plugins      map[string]string
	Enabled      map[string]bool
	Notice       string
	Confirmation string
}

// UIAction is either semantic input or a core lifecycle operation.
type UIAction struct {
	Request      *snoofer.Request
	Kind, Plugin string
	Enable       bool
}

type screen struct {
	ctx                          context.Context
	states                       <-chan ViewState
	actions                      chan<- UIAction
	state                        ViewState
	selected, tab, width, height int
	group                        string
	editing                      bool
	text                         string
	options                      []string
	option                       int
	target                       snoofer.Control
	notice                       string
}
type stateMsg ViewState
type ended struct{}

func (s screen) next() tea.Cmd {
	return func() tea.Msg {
		select {
		case v, ok := <-s.states:
			if ok {
				return stateMsg(v)
			}
		case <-s.ctx.Done():
		}
		return ended{}
	}
}
func (s screen) Init() tea.Cmd { return s.next() }
func (s screen) groups() []string {
	groups := []string{}
	for _, c := range s.state.Controls {
		if !c.SurfaceOnly && !slices.Contains(groups, c.Group) {
			groups = append(groups, c.Group)
		}
	}
	return groups
}
func (s screen) currentGroup() string {
	groups := s.groups()
	if slices.Contains(groups, s.group) || len(groups) == 0 {
		return s.group
	}
	return groups[0]
}
func (s screen) rows() []snoofer.Control {
	group := s.currentGroup()
	rows := []snoofer.Control{}
	for _, c := range s.state.Controls {
		// One-shot recorder transport remains a deck binding, not an Actions section.
		if !c.SurfaceOnly && c.Group == group {
			rows = append(rows, c)
		}
	}
	return rows
}
func (s screen) plugins() []string {
	ids := []string{}
	for id := range s.state.Plugins {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func (s *screen) send(a UIAction) {
	select {
	case s.actions <- a:
		s.notice = ""
	default:
		s.notice = "Controls busy; try again"
	}
}
func (s screen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case ended:
		return s, tea.Quit
	case stateMsg:
		id := ""
		rows := s.rows()
		if s.selected < len(rows) {
			id = rows[s.selected].ID
		}
		previousGroup := s.currentGroup()
		s.state = ViewState(m)
		s.group = previousGroup
		if s.currentGroup() != previousGroup {
			s.selected = 0
		}
		s.group = s.currentGroup()
		if s.tab == 0 {
			for n, c := range s.rows() {
				if c.ID == id {
					s.selected = n
					break
				}
			}
			s.selected = min(s.selected, max(0, len(s.rows())-1))
		}
		return s, s.next()
	case tea.WindowSizeMsg:
		s.width = m.Width
		s.height = m.Height
	case tea.KeyPressMsg:
		k := m.String()
		if s.width < 42 || s.height < 10 {
			if k == "q" || k == "ctrl+c" {
				return s, tea.Quit
			}
			if k == "esc" {
				s.editing = false
			}
			return s, nil
		}
		if s.state.Confirmation != "" {
			if k == "y" {
				s.send(UIAction{Kind: "confirm"})
			}
			if k == "n" || k == "esc" {
				s.send(UIAction{Kind: "cancel"})
			}
			return s, nil
		}
		if s.editing {
			switch k {
			case "esc":
				s.editing = false
			case "enter":
				value := s.text
				if len(s.options) > 0 {
					value = s.options[s.option]
				}
				s.send(UIAction{Request: &snoofer.Request{ID: s.target.ID, Revision: s.target.Revision, Operation: "set", Value: value}})
				s.editing = false
			case "up":
				s.option = max(0, s.option-1)
			case "down":
				s.option = min(max(0, len(s.options)-1), s.option+1)
			case "space", " ":
				if len(s.options) == 0 {
					s.text += " "
				}
			case "backspace":
				r := []rune(s.text)
				if len(r) > 0 {
					s.text = string(r[:len(r)-1])
				}
			default:
				if len(s.options) == 0 && len([]rune(k)) == 1 {
					s.text += k
				}
			}
			return s, nil
		}
		switch k {
		case "q", "ctrl+c":
			return s, tea.Quit
		case "tab", "shift+tab":
			s.tab = 1 - s.tab
			s.selected = 0
		case "left", "right", "[", "]":
			if s.tab == 0 {
				groups := s.groups()
				if len(groups) > 0 {
					step := 1
					if k == "left" || k == "[" {
						step = -1
					}
					index := slices.Index(groups, s.currentGroup())
					s.group = groups[(index+step+len(groups))%len(groups)]
					s.selected = 0
				}
			}
		case "up", "k":
			s.selected = max(0, s.selected-1)
		case "down", "j":
			count := len(s.rows())
			if s.tab == 1 {
				count = len(s.plugins())
			}
			s.selected = min(max(0, count-1), s.selected+1)
		case "home":
			s.selected = 0
		case "end", "pgdown", "pgup":
			count := len(s.rows())
			if s.tab == 1 {
				count = len(s.plugins())
			}
			if k == "end" {
				s.selected = max(0, count-1)
			} else if k == "pgdown" {
				s.selected = min(max(0, count-1), s.selected+max(1, s.height-6))
			} else {
				s.selected = max(0, s.selected-max(1, s.height-6))
			}
		case "r":
			s.send(UIAction{Kind: "retry"})
		case "enter", " ", "space", "+", "-":
			if s.width < 42 || s.height < 10 {
				s.notice = "Enlarge terminal to edit controls"
				break
			}
			if s.tab == 1 {
				ids := s.plugins()
				if s.selected < len(ids) {
					id := ids[s.selected]
					s.send(UIAction{Kind: "selection", Plugin: id, Enable: !s.state.Enabled[id]})
				}
				break
			}
			rows := s.rows()
			if s.selected >= len(rows) {
				break
			}
			c := rows[s.selected]
			if c.EnterOnly && k != "enter" {
				break
			}
			if !c.Available {
				s.notice = "Unavailable: " + c.Status
				break
			}
			if c.Kind == "selection" || c.Kind == "text" {
				s.editing = true
				s.target = c
				s.text = c.Value
				s.options = c.Options
				s.option = 0
				for n, v := range c.Options {
					if v == c.Value {
						s.option = n
					}
				}
			} else {
				r := snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}
				if c.Kind == "numeric" && (k == "+" || k == "-") {
					r.Operation = "adjust"
					r.Delta = 1
					if k == "-" {
						r.Delta = -1
					}
				}
				s.send(UIAction{Request: &r})
			}
		}
	}
	return s, nil
}
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, ansi.Strip(s))
}
func runUI(ctx context.Context, state ViewState, actions chan<- UIAction, states <-chan ViewState, in io.Reader, out io.Writer) error {
	profile := colorprofile.ANSI256
	if os.Getenv("NO_COLOR") != "" {
		profile = colorprofile.Ascii
	}
	_, err := tea.NewProgram(screen{ctx: ctx, state: state, actions: actions, states: states, width: 100, height: 30}, tea.WithColorProfile(profile), tea.WithContext(ctx), tea.WithInput(in), tea.WithOutput(out)).Run()
	return err
}

// stopHost uses a bounded independent cleanup deadline after run cancellation.
func stopHost(h *snoofer.Host) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.Stop(ctx)
}

func confirmation(before, after snoofer.Config) string {
	changes := []string{}
	for id, p := range after.Plugins {
		if before.Plugins[id].Enabled != p.Enabled {
			changes = append(changes, fmt.Sprintf("%s=%t", id, p.Enabled))
		}
	}
	sort.Strings(changes)
	return "Apply " + strings.Join(changes, ", ") + " and restart Snoofer?"
}
