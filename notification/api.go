package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ApiNotification struct {
	logger logr.Logger
	id     string
	url    string
}

func NewApiNotification(ctx context.Context, id string, url string) *ApiNotification {
	return &ApiNotification{
		logger: log.FromContext(ctx),
		id:     id,
		url:    url,
	}
}

func (api *ApiNotification) Notify(body NotificationBody) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(jsonBody)
	resp, err := http.Post(api.url, "application/json", bodyReader)
	if err != nil {
		return err
	}
	api.logger.Info("Notification sent", "id", api.id, "status", resp.Status, "body", string(jsonBody))
	return nil
}
