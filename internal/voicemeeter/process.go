package voicemeeter

import (
	"strings"

	"sound-snoofer/internal/model"
)

func observeElement(list func() ([]string, error)) model.ProcessStatus {
	names, err := list()
	if err != nil {
		return model.ProcessStatus{Error: err.Error()}
	}
	result := model.ProcessStatus{Known: true}
	for _, name := range names {
		if strings.EqualFold(name, "element.exe") {
			result.Running = true
			break
		}
	}
	return result
}
