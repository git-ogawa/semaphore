package sql

import (
	"github.com/Masterminds/squirrel"
	"github.com/semaphoreui/semaphore/db"
)

func (d *SqlDb) GetSlackNotifications(projectID int, params db.RetrieveQueryParams) ([]db.SlackNotification, error) {
	var notifications []db.SlackNotification

	q := squirrel.Select("*").
		From("project__slack_notification").
		Where("project_id=?", projectID).
		OrderBy("name")

	if params.Count > 0 {
		q = q.Limit(uint64(params.Count))
	}
	if params.Offset > 0 {
		q = q.Offset(uint64(params.Offset))
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	_, err = d.selectAll(&notifications, query, args...)
	return notifications, err
}

func (d *SqlDb) GetSlackNotification(projectID int, id int) (db.SlackNotification, error) {
	var n db.SlackNotification

	query, args, err := squirrel.Select("*").
		From("project__slack_notification").
		Where("project_id=? and id=?", projectID, id).
		ToSql()

	if err != nil {
		return n, err
	}

	err = d.selectOne(&n, query, args...)
	return n, err
}

func (d *SqlDb) CreateSlackNotification(n db.SlackNotification) (db.SlackNotification, error) {
	insertID, err := d.insert(
		"id",
		"insert into project__slack_notification (project_id, name, description, token, channel, color, started_message, success_message, error_message) values (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		n.ProjectID, n.Name, n.Description, n.Token, n.Channel, n.Color, n.StartedMessage, n.SuccessMessage, n.ErrorMessage,
	)
	if err != nil {
		return n, err
	}
	n.ID = insertID
	return n, nil
}

func (d *SqlDb) UpdateSlackNotification(n db.SlackNotification) error {
	_, err := d.exec(
		"update project__slack_notification set name=?, description=?, token=?, channel=?, color=?, started_message=?, success_message=?, error_message=? where id=? and project_id=?",
		n.Name, n.Description, n.Token, n.Channel, n.Color, n.StartedMessage, n.SuccessMessage, n.ErrorMessage, n.ID, n.ProjectID,
	)
	return err
}

func (d *SqlDb) DeleteSlackNotification(projectID int, id int) error {
	_, err := d.exec("delete from project__slack_notification where id=? and project_id=?", id, projectID)
	return err
}
