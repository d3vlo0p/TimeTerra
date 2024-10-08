package notification

type NotificationType string

const (
	NotificationTypeApi     NotificationType = "api"
	NotificationTypeMSTeams NotificationType = "msteams"
	NotificationTypeSlack   NotificationType = "slack"
	// NotificationTypeEmail   NotificationType = "email"
)

func (r NotificationType) String() string {
	return string(r)
}
