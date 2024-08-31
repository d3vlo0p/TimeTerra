package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
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

	metadata := ""
	for key, value := range body.Metadata {
		metadata += fmt.Sprintf("\t- %s: %v\n", key, value)
	}

	message := fmt.Sprintf("- Schedule: %s\n- Action: %s\n- Resource: %s\n- Status: %s\n- Metadata: \n%s",
		body.Schedule, body.Action, body.Resource, body.Status, metadata)

	payload := map[string]any{
		"text": s.Title,
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]any{
					"type": "plain_text",
					"text": s.Title,
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{
						"type": "mrkdwn",
						"text": message,
					},
				},
			},
		},
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
