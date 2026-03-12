package state

import (
	"encoding/json"
	"errors"
	"math/rand"
)

const (
	RepeatOff RepeatMode = "off"
	RepeatOne RepeatMode = "one"
	RepeatAll RepeatMode = "all"
)

type RepeatMode string

type QueueItem struct {
	TrackID  string `json:"track_id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Duration int    `json:"duration"`
	CoverRef string `json:"cover_ref"`
}

type PlaybackState struct {
	CurrentTrackID     string      `json:"current_track_id"`
	Queue              []QueueItem `json:"queue"`
	QueuePosition      int         `json:"queue_position"`
	IsPlaying          bool        `json:"is_playing"`
	ShuffleEnabled     bool        `json:"shuffle_enabled"`
	RepeatMode         RepeatMode  `json:"repeat_mode"`
	Volume             float64     `json:"volume"`
	CurrentTimeSeconds int         `json:"current_time_seconds"`
	ContextType        string      `json:"context_type"`
	ContextID          string      `json:"context_id"`
}

func DefaultState() PlaybackState {
	return PlaybackState{
		Queue:         make([]QueueItem, 0),
		QueuePosition: -1,
		RepeatMode:    RepeatOff,
		Volume:        1,
	}
}

func (s PlaybackState) Clone() PlaybackState {
	out := s
	out.Queue = append([]QueueItem(nil), s.Queue...)
	return out
}

func (s *PlaybackState) Normalize() {
	if s.Queue == nil {
		s.Queue = make([]QueueItem, 0)
	}
	if s.RepeatMode != RepeatOff && s.RepeatMode != RepeatOne && s.RepeatMode != RepeatAll {
		s.RepeatMode = RepeatOff
	}
	if s.Volume < 0 {
		s.Volume = 0
	}
	if s.Volume > 1 {
		s.Volume = 1
	}
	if s.CurrentTimeSeconds < 0 {
		s.CurrentTimeSeconds = 0
	}
	if len(s.Queue) == 0 {
		s.QueuePosition = -1
		s.CurrentTrackID = ""
		s.IsPlaying = false
		return
	}
	if s.QueuePosition < 0 {
		s.QueuePosition = 0
	}
	if s.QueuePosition >= len(s.Queue) {
		s.QueuePosition = len(s.Queue) - 1
	}
	if s.CurrentTrackID == "" {
		s.CurrentTrackID = s.Queue[s.QueuePosition].TrackID
	}
}

func (s *PlaybackState) ReplaceQueue(items []QueueItem, start int, contextType, contextID string) {
	s.Queue = sanitizeItems(items)
	s.ContextType = contextType
	s.ContextID = contextID
	if len(s.Queue) == 0 {
		s.QueuePosition = -1
		s.CurrentTrackID = ""
		s.IsPlaying = false
		s.CurrentTimeSeconds = 0
		return
	}
	if start < 0 {
		start = 0
	}
	if start >= len(s.Queue) {
		start = len(s.Queue) - 1
	}
	s.QueuePosition = start
	s.CurrentTrackID = s.Queue[start].TrackID
	s.CurrentTimeSeconds = 0
}

func (s *PlaybackState) Append(items []QueueItem) {
	s.Queue = append(s.Queue, sanitizeItems(items)...)
	s.Normalize()
}

func (s *PlaybackState) RemoveAt(index int) error {
	if index < 0 || index >= len(s.Queue) {
		return errors.New("queue index out of range")
	}
	removedCurrent := index == s.QueuePosition
	s.Queue = append(s.Queue[:index], s.Queue[index+1:]...)
	if len(s.Queue) == 0 {
		s.QueuePosition = -1
		s.CurrentTrackID = ""
		s.IsPlaying = false
		s.CurrentTimeSeconds = 0
		return nil
	}
	if index < s.QueuePosition {
		s.QueuePosition--
	}
	if removedCurrent {
		if s.QueuePosition >= len(s.Queue) {
			s.QueuePosition = len(s.Queue) - 1
		}
		s.CurrentTrackID = s.Queue[s.QueuePosition].TrackID
		s.CurrentTimeSeconds = 0
	}
	s.Normalize()
	return nil
}

func (s *PlaybackState) Move(from, to int) error {
	if from < 0 || from >= len(s.Queue) || to < 0 || to >= len(s.Queue) {
		return errors.New("queue index out of range")
	}
	if from == to {
		return nil
	}
	item := s.Queue[from]
	s.Queue = append(s.Queue[:from], s.Queue[from+1:]...)
	if to >= len(s.Queue) {
		s.Queue = append(s.Queue, item)
	} else {
		s.Queue = append(s.Queue[:to], append([]QueueItem{item}, s.Queue[to:]...)...)
	}
	s.reindexCurrentByTrackID()
	return nil
}

func (s *PlaybackState) ClearQueue() {
	s.Queue = make([]QueueItem, 0)
	s.QueuePosition = -1
	s.CurrentTrackID = ""
	s.IsPlaying = false
	s.CurrentTimeSeconds = 0
	s.ContextType = ""
	s.ContextID = ""
}

func (s *PlaybackState) CurrentItem() (QueueItem, bool) {
	if len(s.Queue) == 0 || s.QueuePosition < 0 || s.QueuePosition >= len(s.Queue) {
		return QueueItem{}, false
	}
	return s.Queue[s.QueuePosition], true
}

func (s *PlaybackState) NextIndex() int {
	if len(s.Queue) == 0 || s.QueuePosition < 0 {
		return -1
	}
	if s.RepeatMode == RepeatOne {
		return s.QueuePosition
	}
	next := s.QueuePosition + 1
	if next < len(s.Queue) {
		return next
	}
	if s.RepeatMode == RepeatAll {
		return 0
	}
	return -1
}

func (s *PlaybackState) PreviousIndex() int {
	if len(s.Queue) == 0 || s.QueuePosition < 0 {
		return -1
	}
	if s.RepeatMode == RepeatOne {
		return s.QueuePosition
	}
	prev := s.QueuePosition - 1
	if prev >= 0 {
		return prev
	}
	if s.RepeatMode == RepeatAll {
		return len(s.Queue) - 1
	}
	return -1
}

func (s *PlaybackState) SetQueuePosition(index int) {
	if len(s.Queue) == 0 || index < 0 || index >= len(s.Queue) {
		return
	}
	s.QueuePosition = index
	s.CurrentTrackID = s.Queue[index].TrackID
	s.CurrentTimeSeconds = 0
}

func (s *PlaybackState) Shuffle(seed int64) {
	if len(s.Queue) < 2 {
		return
	}
	currentID := s.CurrentTrackID
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(s.Queue), func(i, j int) {
		s.Queue[i], s.Queue[j] = s.Queue[j], s.Queue[i]
	})
	s.reindexCurrentByTrackIDWithFallback(currentID)
}

func (s PlaybackState) MarshalJSON() ([]byte, error) {
	type alias PlaybackState
	cp := s.Clone()
	cp.Normalize()
	return json.Marshal(alias(cp))
}

func sanitizeItems(items []QueueItem) []QueueItem {
	out := make([]QueueItem, 0, len(items))
	for _, item := range items {
		if item.TrackID == "" {
			continue
		}
		if item.Duration < 0 {
			item.Duration = 0
		}
		out = append(out, item)
	}
	return out
}

func (s *PlaybackState) reindexCurrentByTrackID() {
	s.reindexCurrentByTrackIDWithFallback(s.CurrentTrackID)
}

func (s *PlaybackState) reindexCurrentByTrackIDWithFallback(trackID string) {
	if len(s.Queue) == 0 {
		s.QueuePosition = -1
		s.CurrentTrackID = ""
		return
	}
	for idx := range s.Queue {
		if s.Queue[idx].TrackID == trackID {
			s.QueuePosition = idx
			s.CurrentTrackID = trackID
			return
		}
	}
	if s.QueuePosition < 0 || s.QueuePosition >= len(s.Queue) {
		s.QueuePosition = 0
	}
	s.CurrentTrackID = s.Queue[s.QueuePosition].TrackID
}
