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

type MSTeamsNotificationConfig struct {
	WebHookUrl string
	CardTitle  string
}

type MSTeamsNotification struct {
	logger logr.Logger
	MSTeamsNotificationConfig
}

func NewMSTeamsNotification(ctx context.Context, config MSTeamsNotificationConfig) *MSTeamsNotification {
	return &MSTeamsNotification{
		logger:                    log.FromContext(ctx),
		MSTeamsNotificationConfig: config,
	}
}

func (n *MSTeamsNotification) Type() NotificationType {
	return NotificationTypeMSTeams
}

type CardContentMSTeams struct {
	Schema  string           `json:"$schema"`
	Type    string           `json:"type"`
	Version string           `json:"version"`
	Body    []map[string]any `json:"body"`
}

type CardAttachmentMSTeams struct {
	ContentType string             `json:"contentType"`
	ContentUrl  string             `json:"contentUrl"`
	Content     CardContentMSTeams `json:"content"`
}

type CardPayloadMSTeams struct {
	Type        string                  `json:"type"`
	Attachments []CardAttachmentMSTeams `json:"attachments"`
}

type CardFactSetMSTeams struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

func (n *MSTeamsNotification) Notify(id string, body NotificationBody) error {
	dataArray := []CardFactSetMSTeams{
		{
			Title: "Schedule",
			Value: body.Schedule,
		},
		{
			Title: "Action",
			Value: body.Action,
		},
		{
			Title: "Resource",
			Value: body.Resource,
		},
		{
			Title: "Status",
			Value: body.Status,
		},
	}

	for key, value := range body.Metadata {
		v, err := json.Marshal(value)
		if err != nil {
			return err
		}
		dataArray = append(dataArray, CardFactSetMSTeams{
			Title: fmt.Sprintf("Metadata %s", key),
			Value: string(v),
		})
	}

	payload := CardPayloadMSTeams{
		Type: "message",
		Attachments: []CardAttachmentMSTeams{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: CardContentMSTeams{
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.5",
					Body: []map[string]any{
						{
							"type":   "TextBlock",
							"size":   "Medium",
							"weight": "Bolder",
							"text":   n.CardTitle,
							"style":  "heading",
							"wrap":   true,
						},
						{
							"type":  "FactSet",
							"facts": dataArray,
						},
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
	resp, err := http.Post(n.WebHookUrl, "application/json", bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	n.logger.Info("Notification sent", "id", id, "status", resp.Status, "body", string(jsonBody))
	return nil
}
