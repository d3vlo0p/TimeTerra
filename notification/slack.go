package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// https://api.slack.com/messaging/webhooks
type SlackNotificationConfig struct {
	WebHookUrl string
	Title      string
}

type SlackNotification struct {
	SlackNotificationConfig
	logger logr.Logger
}

func NewSlackNotification(ctx context.Context, config SlackNotificationConfig) *SlackNotification {
	return &SlackNotification{
		SlackNotificationConfig: config,
		logger:                  log.FromContext(ctx),
	}
}

func (s *SlackNotification) Type() NotificationType {
	return NotificationTypeSlack
}

// POST https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX
// Content-type: application/json
// {
//     "text": "Danny Torrence left a 1 star review for your property.",
//     "blocks": [
// 		{
// 			"type": "header",
// 			"text": {
// 				"type": "plain_text",
// 				"text": "New request"
// 			}
// 		},
// 		{
// 			"type": "section",
// 			"fields": [
// 			{
// 				"type": "mrkdwn",
// 				"text": "*Type:*\nPaid Time Off"
// 			}
// 			]
// 		},
//     ]
// }

func (s *SlackNotification) Notify(id string, body NotificationBody) error {

	datailSection := []map[string]any{
		{
			"type": "mrkdwn",
			"text": fmt.Sprintf("*Schedule:*\n%s", body.Schedule),
		}, {
			"type": "mrkdwn",
			"text": fmt.Sprintf("*Action:*\n%s", body.Action),
		}, {
			"type": "mrkdwn",
			"text": fmt.Sprintf("*Resource:*\n%s", body.Resource),
		}, {
			"type": "mrkdwn",
			"text": fmt.Sprintf("*Status:*\n%s", body.Status),
		},
	}

	blocks := []map[string]any{
		{
			"type": "header",
			"text": map[string]any{
				"type":  "plain_text",
				"text":  s.Title,
				"emoji": true,
			},
		}, {
			"type":   "section",
			"fields": datailSection,
		},
	}

	if len(body.Metadata) > 0 {
		blocks = append(blocks,
			map[string]any{
				"type": "divider",
			},
			map[string]any{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": "*Metadata:*",
				},
			},
		)
		for key, value := range body.Metadata {
			blocks = append(blocks, map[string]any{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s:* %v", key, value),
				},
			})
		}
	}

	payload := map[string]any{
		"text":   s.Title,
		"blocks": blocks,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(jsonBody)
	resp, err := http.Post(s.WebHookUrl, "application/json", bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	s.logger.Info("Notification sent", "id", id, "status", resp.Status, "body", string(jsonBody))
	return nil
}
