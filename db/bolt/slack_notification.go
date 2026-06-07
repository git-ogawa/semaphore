package bolt

import (
	"github.com/semaphoreui/semaphore/db"
)

func (d *BoltDb) GetSlackNotifications(projectID int, params db.RetrieveQueryParams) ([]db.SlackNotification, error) {
	var notifications []db.SlackNotification
	err := d.getObjects(projectID, db.SlackNotificationProps, params, nil, &notifications)
	return notifications, err
}

func (d *BoltDb) GetSlackNotification(projectID int, id int) (db.SlackNotification, error) {
	var n db.SlackNotification
	err := d.getObject(projectID, db.SlackNotificationProps, intObjectID(id), &n)
	return n, err
}

func (d *BoltDb) CreateSlackNotification(n db.SlackNotification) (db.SlackNotification, error) {
	newN, err := d.createObject(n.ProjectID, db.SlackNotificationProps, n)
	if err != nil {
		return n, err
	}
	return newN.(db.SlackNotification), nil
}

func (d *BoltDb) UpdateSlackNotification(n db.SlackNotification) error {
	return d.updateObject(n.ProjectID, db.SlackNotificationProps, n)
}

func (d *BoltDb) DeleteSlackNotification(projectID int, id int) error {
	return d.deleteObject(projectID, db.SlackNotificationProps, intObjectID(id), nil)
}
