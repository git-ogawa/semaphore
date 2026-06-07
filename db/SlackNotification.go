package db

type SlackNotification struct {
	ID              int     `db:"id" json:"id" backup:"-"`
	ProjectID       int     `db:"project_id" json:"project_id" backup:"-"`
	Name            string  `db:"name" json:"name"`
	Description     *string `db:"description" json:"description,omitempty"`
	Token           string  `db:"token" json:"token"`
	Channel         string  `db:"channel" json:"channel"`
	Color           string  `db:"color" json:"color"`
	StartedMessage  *string `db:"started_message" json:"started_message,omitempty"`
	SuccessMessage  *string `db:"success_message" json:"success_message,omitempty"`
	ErrorMessage    *string `db:"error_message" json:"error_message,omitempty"`
}

type SlackNotificationManager interface {
	GetSlackNotifications(projectID int, params RetrieveQueryParams) ([]SlackNotification, error)
	GetSlackNotification(projectID int, id int) (SlackNotification, error)
	CreateSlackNotification(n SlackNotification) (SlackNotification, error)
	UpdateSlackNotification(n SlackNotification) error
	DeleteSlackNotification(projectID int, id int) error
}
