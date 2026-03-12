package service

import (
	"context"
	"errors"
	"sync"
	"time"

	libraryrepo "music-server/internal/modules/library/repository"
	playerstate "music-server/internal/modules/player/state"
)

var ErrQueueIndexOutOfRange = errors.New("queue index out of range")

type Service struct {
	mu       sync.RWMutex
	state    playerstate.PlaybackState
	recorder PlayRecorder
}

type PlayRecorder interface {
	RecordPlayEvent(ctx context.Context, userID string, event libraryrepo.PlayEvent) error
}

func New(recorder PlayRecorder) *Service {
	return &Service{
		state:    playerstate.DefaultState(),
		recorder: recorder,
	}
}

func (s *Service) GetState(_ context.Context) playerstate.PlaybackState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Clone()
}

func (s *Service) SetState(_ context.Context, next playerstate.PlaybackState) playerstate.PlaybackState {
	next.Normalize()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = next
	return s.state.Clone()
}

func (s *Service) ReplaceQueue(
	_ context.Context,
	items []playerstate.QueueItem,
	position int,
	contextType string,
	contextID string,
) playerstate.PlaybackState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ReplaceQueue(items, position, contextType, contextID)
	s.state.IsPlaying = len(s.state.Queue) > 0
	return s.state.Clone()
}

func (s *Service) AppendQueue(_ context.Context, items []playerstate.QueueItem) playerstate.PlaybackState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Append(items)
	return s.state.Clone()
}

func (s *Service) RemoveQueueItem(_ context.Context, index int) (playerstate.PlaybackState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.state.RemoveAt(index); err != nil {
		return playerstate.PlaybackState{}, ErrQueueIndexOutOfRange
	}
	return s.state.Clone(), nil
}

func (s *Service) ClearQueue(_ context.Context) playerstate.PlaybackState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ClearQueue()
	return s.state.Clone()
}

func (s *Service) MoveQueueItem(_ context.Context, from, to int) (playerstate.PlaybackState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.state.Move(from, to); err != nil {
		return playerstate.PlaybackState{}, ErrQueueIndexOutOfRange
	}
	return s.state.Clone(), nil
}

func (s *Service) ShuffleQueue(_ context.Context, enabled bool) playerstate.PlaybackState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ShuffleEnabled = enabled
	if enabled {
		s.state.Shuffle(time.Now().UnixNano())
	}
	return s.state.Clone()
}

func (s *Service) RecordPlayed(ctx context.Context, userID, trackID string, positionSeconds int, contextType, contextID string) error {
	if s.recorder == nil || trackID == "" {
		return nil
	}
	if positionSeconds < 0 {
		positionSeconds = 0
	}
	return s.recorder.RecordPlayEvent(ctx, userID, libraryrepo.PlayEvent{
		TrackID:         trackID,
		PositionSeconds: positionSeconds,
		ContextType:     contextType,
		ContextID:       contextID,
	})
}
