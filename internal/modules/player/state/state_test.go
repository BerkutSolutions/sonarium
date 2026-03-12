package state

import (
	"encoding/json"
	"testing"
)

func TestQueueOperations(t *testing.T) {
	s := DefaultState()
	s.ReplaceQueue([]QueueItem{
		{TrackID: "t1", Title: "one"},
		{TrackID: "t2", Title: "two"},
	}, 0, "album", "a1")
	if len(s.Queue) != 2 || s.CurrentTrackID != "t1" {
		t.Fatalf("unexpected replace state: %+v", s)
	}

	s.Append([]QueueItem{{TrackID: "t3", Title: "three"}})
	if len(s.Queue) != 3 {
		t.Fatalf("append failed: len=%d", len(s.Queue))
	}

	if err := s.Move(2, 1); err != nil {
		t.Fatalf("move failed: %v", err)
	}
	if s.Queue[1].TrackID != "t3" {
		t.Fatalf("move produced wrong order: %+v", s.Queue)
	}

	if err := s.RemoveAt(0); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if len(s.Queue) != 2 {
		t.Fatalf("remove did not change queue length")
	}
}

func TestNextPreviousRepeatModes(t *testing.T) {
	s := DefaultState()
	s.ReplaceQueue([]QueueItem{
		{TrackID: "t1"},
		{TrackID: "t2"},
	}, 1, "tracks", "list")

	if got := s.NextIndex(); got != -1 {
		t.Fatalf("expected -1 for next with repeat off at end, got %d", got)
	}

	s.RepeatMode = RepeatAll
	if got := s.NextIndex(); got != 0 {
		t.Fatalf("expected wrap to 0 for repeat all, got %d", got)
	}

	s.RepeatMode = RepeatOne
	if got := s.NextIndex(); got != s.QueuePosition {
		t.Fatalf("expected same index for repeat one next, got %d", got)
	}
	if got := s.PreviousIndex(); got != s.QueuePosition {
		t.Fatalf("expected same index for repeat one previous, got %d", got)
	}
}

func TestShuffleKeepsCurrentTrack(t *testing.T) {
	s := DefaultState()
	s.ReplaceQueue([]QueueItem{
		{TrackID: "t1"},
		{TrackID: "t2"},
		{TrackID: "t3"},
		{TrackID: "t4"},
	}, 2, "album", "a1")

	current := s.CurrentTrackID
	s.Shuffle(42)

	if s.CurrentTrackID != current {
		t.Fatalf("shuffle changed current track id: %s -> %s", current, s.CurrentTrackID)
	}
	if s.Queue[s.QueuePosition].TrackID != current {
		t.Fatalf("queue position not aligned with current track after shuffle")
	}
}

func TestPlaybackStateSerialization(t *testing.T) {
	s := PlaybackState{
		Queue:         []QueueItem{{TrackID: "t1", Duration: 10}},
		QueuePosition: 0,
		Volume:        2,
		RepeatMode:    "unknown",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed PlaybackState
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	parsed.Normalize()
	if parsed.Volume != 1 {
		t.Fatalf("expected normalized volume=1, got %f", parsed.Volume)
	}
	if parsed.RepeatMode != RepeatOff {
		t.Fatalf("expected repeat off, got %s", parsed.RepeatMode)
	}
}
