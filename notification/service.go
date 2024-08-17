package notification

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NotificationBody struct {
	Schedule string
	Action   string
	Resource string
	Message  string
	Status   string
}

type Recipient interface {
	Notify(body NotificationBody) error
}

type NotificationService struct {
	ch         chan NotificationBody
	recipients map[string]map[string]Recipient
	logger     logr.Logger
}

func New() *NotificationService {
	return &NotificationService{
		ch: make(chan NotificationBody),
	}
}

func (s *NotificationService) RemoveRecipient(id string) {
	if s.recipients == nil {
		return
	}
	// remove recipient from all schedules
	for _, recipients := range s.recipients {
		delete(recipients, id)
	}
}

func (s *NotificationService) AddRecipientToSchedule(schedule string, id string, recipient Recipient) {
	if s.recipients == nil {
		s.recipients = make(map[string]map[string]Recipient)
	}
	if s.recipients[schedule] == nil {
		s.recipients[schedule] = make(map[string]Recipient)
	}
	s.recipients[schedule][id] = recipient
}

func (s *NotificationService) Send(body NotificationBody) {
	s.ch <- body
}

func (s *NotificationService) Start(ctx context.Context) error {
	s.logger = log.FromContext(ctx)
	var wg sync.WaitGroup
	// start notification service
	s.logger.Info("Starting notification service")
	wg.Add(1)
	go func() {
		s.logger.Info("Notification service started")
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
				recipient.Notify(body)
			}
		}
		s.logger.Info("Notification service stopped")
	}()
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
