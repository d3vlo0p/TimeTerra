package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ApiNotificationConfig struct {
	Url    string
	Method string
}

type ApiNotification struct {
	logger logr.Logger
	ApiNotificationConfig
}

func NewApiNotification(ctx context.Context, config ApiNotificationConfig) *ApiNotification {
	return &ApiNotification{
		logger:                log.FromContext(ctx),
		ApiNotificationConfig: config,
	}
}

func (n *ApiNotification) Type() NotificationType {
	return NotificationTypeApi
}

func (n *ApiNotification) Notify(id string, body NotificationBody) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(jsonBody)
	req, err := http.NewRequest(strings.ToUpper(n.Method), n.Url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api notification returned non-success status: %s", resp.Status)
	}
	n.logger.Info("Notification sent", "id", id, "status", resp.Status, "body", string(jsonBody))
	return nil
}
