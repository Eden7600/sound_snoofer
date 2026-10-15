package app

import (
	"context"
	"errors"
	"fmt"

	"sound-snoofer/snoofer"
)

// coreHandler preserves the published startup target through the second actor queue.
func coreHandler(providerCtx context.Context, actions chan<- UIAction, observed startupState, enable bool) func(context.Context, snoofer.Request) error {
	return func(ctx context.Context, r snoofer.Request) error {
		if err := providerCtx.Err(); err != nil {
			return err
		}
		action := UIAction{Kind: "open"}
		if r.ID == "core.startup" {
			action = UIAction{Kind: "startup", Enable: enable, startupExpected: observed}
		}
		action.complete = r.ClaimCompletion(providerCtx)
		select {
		case actions <- action:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("controls busy")
		}
	}
}

func applyStartup(ctx context.Context, action UIAction, live bool, command string, read func() (startupState, error), apply func(bool, string) error) snoofer.Outcome {
	if ctx.Err() != nil {
		return snoofer.Outcome{Status: snoofer.Unknown, Detail: "Host stopped before startup change"}
	}
	if !live {
		return snoofer.Outcome{Status: snoofer.Rejected, Detail: "Startup unavailable in preview"}
	}
	observed, err := read()
	if err != nil {
		return lifecycleFailure(err)
	}
	if observed != action.startupExpected {
		return snoofer.Outcome{Status: snoofer.Rejected, Detail: "Startup registration changed; try again"}
	}
	if err := apply(action.Enable, command); err != nil {
		return lifecycleFailure(err)
	}
	return snoofer.Outcome{Status: snoofer.Processed, Detail: "Startup updated"}
}

func lifecycleFailure(err error) snoofer.Outcome {
	status := snoofer.Failed
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		status = snoofer.Unknown
	}
	return snoofer.Outcome{Status: status, Detail: err.Error()}
}
