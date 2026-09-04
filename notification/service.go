package notification

import (
	"context"
	"sync"
	"time"

	"github.com/d3vlo0p/TimeTerra/monitoring"
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
	Notify(id string, body NotificationBody) error
	Type() NotificationType
}

type NotificationService struct {
	mu         sync.RWMutex
	ch         chan NotificationBody
	recipients map[string]map[string]Recipient
	logger     logr.Logger
	threads    uint
	closed     bool
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		ch:         make(chan NotificationBody, 256),
		recipients: make(map[string]map[string]Recipient),
		logger:     log.Log.WithName("notification"),
		threads:    1,
	}
}

func (s *NotificationService) RemoveRecipient(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.recipients == nil {
		return
	}
	// remove recipient from all schedules
	for scheduleName := range s.recipients {
		s.logger.Info("Removing recipient", "id", id, "schedule", scheduleName)
		recipient, ok := s.recipients[scheduleName][id]
		if !ok {
			continue
		}
		monitoring.TimeTerraNotificationPolicies.WithLabelValues(recipient.Type().String(), scheduleName).Dec()
		delete(s.recipients[scheduleName], id)
	}
}

func (s *NotificationService) AddRecipientToSchedule(schedule string, id string, recipient Recipient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.recipients == nil {
		s.recipients = make(map[string]map[string]Recipient)
	}
	if s.recipients[schedule] == nil {
		s.recipients[schedule] = make(map[string]Recipient)
	}
	// add recipient to schedule
	s.logger.Info("Adding recipient", "id", id, "schedule", schedule)
	s.recipients[schedule][id] = recipient
	monitoring.TimeTerraNotificationPolicies.WithLabelValues(recipient.Type().String(), schedule).Inc()
}

func (s *NotificationService) Send(body NotificationBody) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		s.logger.Info("Notification service is stopped, dropping notification", "schedule", body.Schedule, "action", body.Action)
		return
	}

	select {
	case s.ch <- body:
	default:
		s.logger.Info("Notification channel is full, dropping notification to prevent blocking", "schedule", body.Schedule, "action", body.Action)
	}
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

		s.mu.RLock()
		recipientsCopy := make(map[string]Recipient)
		if schedRecipients, ok := s.recipients[body.Schedule]; ok {
			for id, rec := range schedRecipients {
				recipientsCopy[id] = rec
			}
		}
		s.mu.RUnlock()

		for id, recipient := range recipientsCopy {
			start := time.Now()
			status := "success"
			err := recipient.Notify(id, body)
			if err != nil {
				s.logger.Error(err, "Failed to notify recipient")
				status = "failed"
			}
			monitoring.TimeTerraNotificationLatency.WithLabelValues(recipient.Type().String(), body.Schedule, body.Action, body.Resource, status).Observe(time.Since(start).Seconds())
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

		s.mu.Lock()
		s.closed = true
		close(s.ch)
		s.mu.Unlock()

		s.logger.Info("Waiting for notification service to stop")
		// wait for notification service to stop
		wg.Wait()
		close(stopSignal)
	}()
	// wait for stop signal
	<-stopSignal
	return nil
}
