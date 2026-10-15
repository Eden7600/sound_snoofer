package nowplaying

import (
	"errors"
	"time"

	"sound-snoofer/snoofer"
)

// A rating reservation belongs to the song, even while focus or eligibility changes.
type ratingTarget struct {
	key, source, instance, track string
	generation                   uint64
}

func ratingTargetOf(s session) ratingTarget {
	return ratingTarget{s.Key, s.Source, s.Instance, s.Track, s.Generation}
}

type ratingState struct {
	target  ratingTarget
	want    string
	at      time.Time
	pending bool
	failure string
}

func knownRating(rating string) bool {
	return rating == "none" || rating == "like" || rating == "dislike"
}

func (w *worker) rate(captured session, want string, now time.Time) snoofer.Outcome {
	if !w.services.Live {
		return reject("Media commands unavailable in preview")
	}
	current, ok := w.find(captured.Key)
	target := ratingTargetOf(captured)
	if !ok || ratingTargetOf(current) != target || !w.targetAvailable(captured) {
		return reject("Media session changed or unavailable; try again")
	}
	if prior := w.ratings[captured.Key]; prior.target == target && prior.pending {
		return reject("Rating pending")
	}
	if current.Source == "windows" || current.Track == "" || !current.CanRate || !knownRating(current.Rating) || !captured.CanRate || !knownRating(want) {
		return reject("Rating unavailable")
	}
	pending := ratingState{target: target, want: want, at: now, pending: true}
	err := w.send(current.Source, browserCommand{Type: "command", ID: current.ID, Instance: current.Instance, Generation: current.Generation, Op: "rate", Track: current.Track, Rating: want})
	if errors.Is(err, errBrowserChanged) {
		return reject(err.Error())
	}
	if err != nil {
		pending.failure = err.Error()
		w.ratings[current.Key] = pending
		return snoofer.Outcome{Status: snoofer.Unknown, Detail: err.Error()}
	}
	w.ratings[current.Key] = pending
	return snoofer.Outcome{Status: snoofer.Submitted, Detail: "Browser request submitted"}
}

func (w *worker) observeRatings(now time.Time) {
	for key, pending := range w.ratings {
		current, ok := w.find(key)
		if !ok || ratingTargetOf(current) != pending.target || !w.targetAvailable(current) {
			delete(w.ratings, key)
			continue
		}
		if !pending.pending {
			continue
		}
		if current.Rating == pending.want {
			delete(w.ratings, key)
		} else if now.Sub(pending.at) > w.timing.observe {
			pending.pending, pending.failure = false, "Rating not confirmed"
			w.ratings[key] = pending
		}
	}
}

func (w *worker) ratingControl(s session, rating, label string) snoofer.Control {
	value := "N/A"
	if knownRating(s.Rating) {
		value = snoofer.OnOff(s.Rating == rating)
	}
	status := ""
	if pending := w.ratings[s.Key]; pending.target == ratingTargetOf(s) {
		status = pending.failure
		if status == "" && pending.pending {
			status = "Pending"
		}
	}
	return snoofer.Control{ID: "nowplaying." + rating, Label: label + " Track", ShortLabel: label, Group: "Now Playing", Kind: "command", Icon: "media-" + rating,
		Value: value, Status: status, Operations: []string{"press"}, Epoch: w.focusGeneration,
		Hidden: s.Rating == "" || w.settings.DeckMediaOff, Available: w.targetAvailable(s) && s.CanRate && knownRating(s.Rating)}
}
