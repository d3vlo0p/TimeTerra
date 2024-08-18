package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ApiNotificationConfig struct {
	Url    string
	Method string
}

type ApiNotification struct {
	logger logr.Logger
	id     string
	ApiNotificationConfig
}

func NewApiNotification(ctx context.Context, id string, config ApiNotificationConfig) *ApiNotification {
	return &ApiNotification{
		logger:                log.FromContext(ctx),
		id:                    id,
		ApiNotificationConfig: config,
	}
}

func (api *ApiNotification) Notify(body NotificationBody) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(jsonBody)
	req, err := http.NewRequest(strings.ToUpper(api.Method), api.Url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	api.logger.Info("Notification sent", "id", api.id, "status", resp.Status, "body", string(jsonBody))
	return nil
}
