package api

import "github.com/y08lin4/lyra-image-workbench/internal/activitylog"

func recordActivity(recorder activitylog.Recorder, input activitylog.EntryInput) {
	if recorder == nil {
		return
	}
	_, _ = recorder.Append(input)
}
