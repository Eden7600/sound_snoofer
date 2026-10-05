package voicemeeter

import (
	"strings"

	"sound-snoofer/internal/model"
)

func observeProcess(names []string, err error, want string) model.ProcessStatus {
	if err != nil {
		return model.ProcessStatus{Error: err.Error()}
	}
	result := model.ProcessStatus{Known: true}
	for _, name := range names {
		if strings.EqualFold(name, want) {
			result.Running = true
			break
		}
	}
	return result
}
