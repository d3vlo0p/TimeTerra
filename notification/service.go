package notification

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NotificationBody struct {
	Schedule string         `json:"schedule"`
	Action   string         `json:"action"`
	Resource string         `json:"resource"`
	Status   string         `json:"status"`
	Metadata map[string]any `json:"metadata"`
}

type Recipient interface {
	Notify(body NotificationBody) error
}

type NotificationService struct {
	ch         chan NotificationBody
	recipients map[string]map[string]Recipient
	logger     logr.Logger
	threads    uint
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		ch:      make(chan NotificationBody),
		threads: 3,
	}
}

func (s *NotificationService) RemoveRecipient(id string) {
	if s.recipients == nil {
		return
	}
	// remove recipient from all schedules
	for scheduleName := range s.recipients {
		s.logger.Info("Removing recipient", "id", id, "schedule", scheduleName)
		delete(s.recipients[scheduleName], id)
	}
}

func (s *NotificationService) AddRecipientToSchedule(schedule string, id string, recipient Recipient) {
	if s.recipients == nil {
		s.recipients = make(map[string]map[string]Recipient)
	}
	if s.recipients[schedule] == nil {
		s.recipients[schedule] = make(map[string]Recipient)
	}
	// add recipient to schedule
	s.logger.Info("Adding recipient", "id", id, "schedule", schedule)
	s.recipients[schedule][id] = recipient
}

func (s *NotificationService) Send(body NotificationBody) {
	s.ch <- body
}

func (s *NotificationService) run(i uint, wg *sync.WaitGroup) {
	s.logger.Info("Notification service started", "thread", i)
	defer wg.Done()
	// loop until channel is closed
	for {
		body, ok := <-s.ch
		if !ok {
			break
		}
		if _, ok := s.recipients[body.Schedule]; !ok {
			continue
		}
		for _, recipient := range s.recipients[body.Schedule] {
			err := recipient.Notify(body)
			if err != nil {
				s.logger.Error(err, "Failed to notify recipient")
			}
		}
	}
	s.logger.Info("Notification service stopped", "thread", i)
}

func (s *NotificationService) Start(ctx context.Context) error {
	s.logger = log.FromContext(ctx)
	var wg sync.WaitGroup

	// start notification service
	s.logger.Info("Starting notification service")
	for i := uint(0); i < s.threads; i++ {
		wg.Add(1)
		go s.run(i, &wg)
	}

	stopSignal := make(chan struct{})
	go func() {
		<-ctx.Done()
		s.logger.Info("Closing notification service channel")
		close(s.ch)
		s.logger.Info("Waiting for notification service to stop")
		// wait for notification service to stop
		wg.Wait()
		close(stopSignal)
	}()
	// wait for stop signal
	<-stopSignal
	return nil
}
