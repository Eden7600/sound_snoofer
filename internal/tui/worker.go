package tui

import (
	"context"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
)

type State = control.State
type Action = control.Action
type ActionKind = control.ActionKind
type Dependencies = control.Dependencies
type Client = control.Client
type settingEdit = control.SettingEdit

const maxDeferredEdits = 64
const (
	toggleLive ActionKind = iota
	reload
	refresh
	editRule
	resetChoices
	startRecording
	stopRecording
	playSnippet
)
const (
	noticeError   = control.NoticeError
	noticePending = control.NoticePending
	noticeSuccess = control.NoticeSuccess
)

var ToggleLive = control.ToggleLive
var Reload = control.Reload
var Refresh = control.Refresh
var editIntent = control.EditIntent
var editBatch = control.EditBatch
var editsRow = control.EditsRow

func work(ctx context.Context, cfg config.Config, path, dll string, live bool, deps Dependencies, actions <-chan Action, states chan State, done chan struct{}) {
	control.Work(ctx, cfg, path, dll, live, deps, actions, states, done)
}
