package notification

type NotificationType string

const (
	NotificationTypeApi     NotificationType = "api"
	NotificationTypeMSTeams NotificationType = "msteams"
)

func (r NotificationType) String() string {
	return string(r)
}
