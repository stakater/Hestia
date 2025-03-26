package scheduler

import (
	"context"
	"github.com/robfig/cron/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sync"
)

type CronEntry struct {
	Cron string
	ID   cron.EntryID
}
type CronEvent = event.TypedGenericEvent[client.ObjectKey]
type CronScheduler struct {
	Events   chan CronEvent
	cron     *cron.Cron
	registry map[client.ObjectKey]CronEntry
	mu       sync.Mutex
}

func NewCronScheduler() *CronScheduler {
	return &CronScheduler{
		Events:   make(chan CronEvent),
		cron:     cron.New(cron.WithSeconds()),
		registry: make(map[client.ObjectKey]CronEntry),
		mu:       sync.Mutex{},
	}
}

func (s *CronScheduler) Start(ctx context.Context) error {
	s.cron.Start()

	<-ctx.Done()
	c := s.cron.Stop()
	<-c.Done()

	return nil
}

func (s *CronScheduler) Add(cronStr string, owner client.ObjectKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.registry[owner]; ok {
		if entry.Cron != cronStr {
			s.cron.Remove(entry.ID)
		} else {
			return nil
		}
	}

	id, err := s.cron.AddFunc(cronStr, func() {
		s.Events <- CronEvent{
			Object: owner,
		}
	})

	if err != nil {
		return err
	}

	s.registry[owner] = CronEntry{
		Cron: cronStr,
		ID:   id,
	}

	return nil
}

func (s *CronScheduler) Remove(id cron.EntryID) {
	s.cron.Remove(id)
}
