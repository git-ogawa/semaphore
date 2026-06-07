package projects

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/services/tasks"
	log "github.com/sirupsen/logrus"
)

func GetSlackNotifications(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	notifications, err := helpers.Store(r).GetSlackNotifications(project.ID, db.RetrieveQueryParams{})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, notifications)
}

func AddSlackNotification(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)

	var n db.SlackNotification
	if !helpers.Bind(w, r, &n) {
		return
	}
	n.ProjectID = project.ID

	newN, err := helpers.Store(r).CreateSlackNotification(n)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, newN)
}

func GetSlackNotification(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	id, err := strconv.Atoi(mux.Vars(r)["slack_notification_id"])
	if err != nil {
		helpers.WriteErrorStatus(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	n, err := helpers.Store(r).GetSlackNotification(project.ID, id)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, n)
}

func UpdateSlackNotification(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	id, err := strconv.Atoi(mux.Vars(r)["slack_notification_id"])
	if err != nil {
		helpers.WriteErrorStatus(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var n db.SlackNotification
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		log.Error(err)
		helpers.WriteErrorStatus(w, "Bad request", http.StatusBadRequest)
		return
	}
	n.ID = id
	n.ProjectID = project.ID

	if err := helpers.Store(r).UpdateSlackNotification(n); err != nil {
		helpers.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func TestSlackNotification(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	id, err := strconv.Atoi(mux.Vars(r)["slack_notification_id"])
	if err != nil {
		helpers.WriteErrorStatus(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	n, err := helpers.Store(r).GetSlackNotification(project.ID, id)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	if err := tasks.SendSlackTestNotification(n); err != nil {
		helpers.WriteErrorStatus(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func DeleteSlackNotification(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	id, err := strconv.Atoi(mux.Vars(r)["slack_notification_id"])
	if err != nil {
		helpers.WriteErrorStatus(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := helpers.Store(r).DeleteSlackNotification(project.ID, id); err != nil {
		helpers.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
