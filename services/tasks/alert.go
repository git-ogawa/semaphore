package tasks

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/task_logger"
	"github.com/semaphoreui/semaphore/util"
	"github.com/semaphoreui/semaphore/util/mailer"
)

//go:embed templates/*.tmpl
var templates embed.FS

// Alert represents an alert that will be templated and sent to the appropriate service
type Alert struct {
	Name   string
	Author string
	Color  string
	Task   alertTask
	Chat   alertChat
}

type alertTask struct {
	ID      string
	URL     string
	Result  string
	Desc    string
	Version string
}

type alertChat struct {
	ID string
}

func (t *TaskRunner) sendMailAlert() {
	if !util.Config.EmailAlert || !t.alert {
		return
	}

	body := bytes.NewBufferString("")
	author, version := t.alertInfos()

	alert := Alert{
		Name:   t.Template.Name,
		Author: author,
		Color:  t.alertColor("email"),
		Task: alertTask{
			ID:      strconv.Itoa(t.Task.ID),
			URL:     t.taskLink(),
			Result:  t.Task.Status.Format(),
			Version: version,
			Desc:    t.Task.Message,
		},
	}

	tpl, err := htmltemplate.ParseFS(templates, "templates/email.tmpl")

	if err != nil {
		t.Log("Can't parse email alert template!")
		panic(err)
	}

	if err := tpl.Execute(body, alert); err != nil {
		t.Log("Can't generate email alert template!")
		panic(err)
	}

	if body.Len() == 0 {
		t.Log("Buffer for email alert is empty")
		return
	}

	for _, uid := range t.users {
		user, err := t.pool.store.GetUser(uid)

		if err != nil {
			util.LogError(err)
			continue
		}

		if !user.Alert {
			continue
		}

		t.Logf("Attempting to send email alert to %s", user.Email)

		str := body.String()
		if err := mailer.Send(
			util.Config.EmailSecure,
			util.Config.EmailTls,
			util.Config.EmailHost,
			util.Config.EmailPort,
			util.Config.EmailUsername,
			util.Config.EmailPassword,
			util.Config.EmailSender,
			user.Email,
			fmt.Sprintf("Task '%s' failed", t.Template.Name),
			str,
		); err != nil {
			util.LogError(err)
			continue
		}

		t.Logf("Sent successfully email alert to %s", user.Email)
	}
}

func (t *TaskRunner) sendTelegramAlert() {
	if !util.Config.TelegramAlert || !t.alert {
		return
	}

	if t.Template.SuppressSuccessAlerts && t.Task.Status == task_logger.TaskSuccessStatus {
		return
	}

	chatID := util.Config.TelegramChat
	if t.alertChat != nil && *t.alertChat != "" {
		chatID = *t.alertChat
	}

	if chatID == "" {
		return
	}

	body := bytes.NewBufferString("")
	author, version := t.alertInfos()

	alert := Alert{
		Name:   t.Template.Name,
		Author: author,
		Color:  t.alertColor("telegram"),
		Task: alertTask{
			ID:      strconv.Itoa(t.Task.ID),
			URL:     t.taskLink(),
			Result:  t.Task.Status.Format(),
			Version: version,
			Desc:    t.Task.Message,
		},
		Chat: alertChat{
			ID: chatID,
		},
	}

	tpl, err := template.ParseFS(templates, "templates/telegram.tmpl")

	if err != nil {
		t.Log("Can't parse telegram alert template!")
		panic(err)
	}

	if err := tpl.Execute(body, alert); err != nil {
		t.Log("Can't generate telegram alert template!")
		panic(err)
	}

	if body.Len() == 0 {
		t.Log("Buffer for telegram alert is empty")
		return
	}

	t.Log("Attempting to send telegram alert")

	resp, err := http.Post(
		fmt.Sprintf(
			"https://api.telegram.org/bot%s/sendMessage",
			util.Config.TelegramToken,
		),
		"application/json",
		body,
	)

	if err != nil {
		t.Log("Can't send telegram alert! Error: " + err.Error())
	} else if resp.StatusCode != 200 {
		t.Log("Can't send telegram alert! Response code: " + strconv.Itoa(resp.StatusCode))
	} else {
		t.Log("Sent successfully telegram alert")
	}

	if resp != nil {
		defer resp.Body.Close() //nolint:errcheck
	}
}

func (t *TaskRunner) sendSlackAlert() {
	if !util.Config.SlackAlert || !t.alert {
		return
	}

	if t.Template.SuppressSuccessAlerts && t.Task.Status == task_logger.TaskSuccessStatus {
		return
	}

	if t.Template.SuppressFailureAlerts && t.Task.Status == task_logger.TaskFailStatus {
		return
	}

	var notifID *int
	switch t.Task.Status {
	case task_logger.TaskSuccessStatus:
		notifID = t.Template.SlackNotificationSuccessID
	case task_logger.TaskFailStatus:
		notifID = t.Template.SlackNotificationFailureID
	case task_logger.TaskRunningStatus:
		notifID = t.Template.SlackNotificationStartedID
	}

	if notifID != nil {
		n, err := t.pool.store.GetSlackNotification(t.Template.ProjectID, *notifID)
		if err != nil {
			t.Log("Can't get slack notification template! Error: " + err.Error())
			return
		}
		t.sendSlackAlertViaAPI(n)
		return
	}

	if util.Config.SlackToken != "" && util.Config.SlackChannel != "" {
		t.sendSlackAlertViaAPIGlobal()
		return
	}

	t.sendSlackAlertViaWebhook()
}

func (t *TaskRunner) slackMessage(n *db.SlackNotification) string {
	var customMsg *string
	if n != nil {
		switch t.Task.Status {
		case task_logger.TaskSuccessStatus:
			customMsg = n.SuccessMessage
		case task_logger.TaskFailStatus:
			customMsg = n.ErrorMessage
		case task_logger.TaskRunningStatus:
			customMsg = n.StartedMessage
		}
	}
	if customMsg != nil && *customMsg != "" {
		tmplStr := *customMsg
		tmpl, err := template.New("slack_msg").Parse(tmplStr)
		if err != nil {
			t.Log("Can't parse slack message template: " + err.Error())
			return fmt.Sprintf("Job #%d '%s' %s: %s", t.Task.ID, t.Template.Name, t.Task.Status.Format(), t.taskLink())
		}
		data := map[string]any{
			"JobID":   t.Task.ID,
			"Name":    t.Template.Name,
			"Status":  t.Task.Status.Format(),
			"URL":     t.taskLink(),
			"Author":  "",
			"Version": "",
		}
		author, version := t.alertInfos()
		data["Author"] = author
		data["Version"] = version

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			t.Log("Can't execute slack message template: " + err.Error())
			return fmt.Sprintf("Job #%d '%s' %s: %s", t.Task.ID, t.Template.Name, t.Task.Status.Format(), t.taskLink())
		}
		return buf.String()
	}
	return fmt.Sprintf("Job #%d '%s' %s: %s", t.Task.ID, t.Template.Name, t.Task.Status.Format(), t.taskLink())
}

func (t *TaskRunner) sendSlackAlertViaAPIGlobal() {
	n := db.SlackNotification{
		Token:   util.Config.SlackToken,
		Channel: util.Config.SlackChannel,
		Color:   t.alertColor("slack"),
	}
	t.sendSlackAlertViaAPI(n)
}

func (t *TaskRunner) sendSlackAlertViaAPI(n db.SlackNotification) {
	color := n.Color
	msg := t.slackMessage(&n)

	for _, recipient := range strings.Split(n.Channel, ",") {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			continue
		}

		channel := recipient
		threadTs := ""
		if idx := strings.Index(recipient, ":"); idx >= 0 {
			channel = recipient[:idx]
			threadTs = recipient[idx+1:]
		}
		if strings.HasPrefix(channel, "#") {
			channel = channel[1:]
		}

		var payload string
		if color != "" {
			if threadTs != "" {
				payload = fmt.Sprintf(
					`{"channel":%q,"thread_ts":%q,"as_user":true,"attachments":[{"color":%q,"text":%q}]}`,
					channel, threadTs, color, msg,
				)
			} else {
				payload = fmt.Sprintf(
					`{"channel":%q,"as_user":true,"attachments":[{"color":%q,"text":%q}]}`,
					channel, color, msg,
				)
			}
		} else {
			if threadTs != "" {
				payload = fmt.Sprintf(
					`{"channel":%q,"thread_ts":%q,"as_user":true,"text":%q}`,
					channel, threadTs, msg,
				)
			} else {
				payload = fmt.Sprintf(
					`{"channel":%q,"as_user":true,"text":%q}`,
					channel, msg,
				)
			}
		}

		req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBufferString(payload))
		if err != nil {
			t.Log("Can't create slack API request! Error: " + err.Error())
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+n.Token)

		t.Logf("Attempting to send slack alert to %s", channel)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Log("Can't send slack alert! Error: " + err.Error())
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck

		if resp.StatusCode != 200 {
			t.Log("Can't send slack alert! Response code: " + strconv.Itoa(resp.StatusCode))
		} else {
			var result slackResponse
			if err := json.Unmarshal(respBody, &result); err == nil && !result.OK {
				t.Logf("Slack API error: %s", result.Error)
			} else {
				t.Logf("Sent successfully slack alert to %s", channel)
			}
		}
	}
}

func SendSlackTestNotification(n db.SlackNotification) error {
	msg := "This is a test notification from Semaphore"
	for _, recipient := range strings.Split(n.Channel, ",") {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			continue
		}
		channel := recipient
		if strings.HasPrefix(channel, "#") {
			channel = channel[1:]
		}

		var payload string
		if n.Color != "" {
			payload = fmt.Sprintf(
				`{"channel":%q,"as_user":true,"attachments":[{"color":%q,"text":%q}]}`,
				channel, n.Color, msg,
			)
		} else {
			payload = fmt.Sprintf(
				`{"channel":%q,"as_user":true,"text":%q}`,
				channel, msg,
			)
		}

		req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBufferString(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+n.Token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck

		if resp.StatusCode != 200 {
			return fmt.Errorf("slack API returned %d", resp.StatusCode)
		}

		var result slackResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse slack response")
		}
		if !result.OK {
			return fmt.Errorf("slack error: %s", result.Error)
		}
	}
	return nil
}

type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func (t *TaskRunner) sendSlackAlertViaWebhook() {
	body := bytes.NewBufferString("")
	author, version := t.alertInfos()

	alert := Alert{
		Name:   t.Template.Name,
		Author: author,
		Color:  t.alertColor("slack"),
		Task: alertTask{
			ID:      strconv.Itoa(t.Task.ID),
			URL:     t.taskLink(),
			Result:  t.Task.Status.Format(),
			Version: version,
			Desc:    t.Task.Message,
		},
	}

	tpl, err := template.ParseFS(templates, "templates/slack.tmpl")

	if err != nil {
		t.Log("Can't parse slack alert template!")
		panic(err)
	}

	if err := tpl.Execute(body, alert); err != nil {
		t.Log("Can't generate slack alert template!")
		panic(err)
	}

	if body.Len() == 0 {
		t.Log("Buffer for slack alert is empty")
		return
	}

	slackUrl := util.Config.SlackUrl
	if t.alertSlackUrl != nil && *t.alertSlackUrl != "" {
		slackUrl = *t.alertSlackUrl
	}

	t.Log("Attempting to send slack alert via webhook")

	resp, err := http.Post(
		slackUrl,
		"application/json",
		body,
	)

	if err != nil {
		t.Log("Can't send slack alert! Error: " + err.Error())
	} else if resp.StatusCode != 200 {
		t.Log("Can't send slack alert! Response code: " + strconv.Itoa(resp.StatusCode))
	} else {
		t.Log("Sent successfully slack alert")
	}

	if resp != nil {
		defer resp.Body.Close() //nolint:errcheck
	}
}

func (t *TaskRunner) sendRocketChatAlert() {
	if !util.Config.RocketChatAlert || !t.alert {
		return
	}

	if t.Template.SuppressSuccessAlerts && t.Task.Status == task_logger.TaskSuccessStatus {
		return
	}

	body := bytes.NewBufferString("")
	author, version := t.alertInfos()

	alert := Alert{
		Name:   t.Template.Name,
		Author: author,
		Color:  t.alertColor("rocketchat"),
		Task: alertTask{
			ID:      strconv.Itoa(t.Task.ID),
			URL:     t.taskLink(),
			Result:  t.Task.Status.Format(),
			Version: version,
			Desc:    t.Task.Message,
		},
	}

	tpl, err := template.ParseFS(templates, "templates/rocketchat.tmpl")

	if err != nil {
		t.Log("Can't parse rocketchat alert template!")
		panic(err)
	}

	if err := tpl.Execute(body, alert); err != nil {
		t.Log("Can't generate rocketchat alert template!")
		panic(err)
	}

	if body.Len() == 0 {
		t.Log("Buffer for rocketchat alert is empty")
		return
	}

	t.Log("Attempting to send rocketchat alert")

	resp, err := http.Post(
		util.Config.RocketChatUrl,
		"application/json",
		body,
	)

	if err != nil {
		t.Log("Can't send rocketchat alert! Error: " + err.Error())
	} else if resp.StatusCode != 200 {
		t.Log("Can't send rocketchat alert! Response code: " + strconv.Itoa(resp.StatusCode))
	} else {
		t.Log("Sent successfully rocketchat alert")
	}
	if resp != nil {
		defer resp.Body.Close() //nolint:errcheck
	}
}

func (t *TaskRunner) sendMicrosoftTeamsAlert() {
	if !util.Config.MicrosoftTeamsAlert || !t.alert {
		return
	}

	if t.Template.SuppressSuccessAlerts && t.Task.Status == task_logger.TaskSuccessStatus {
		return
	}

	body := bytes.NewBufferString("")
	author, version := t.alertInfos()

	alert := Alert{
		Name:   t.Template.Name,
		Author: author,
		Color:  t.alertColor("microsoft-teams"),
		Task: alertTask{
			ID:      strconv.Itoa(t.Task.ID),
			URL:     t.taskLink(),
			Result:  t.Task.Status.Format(),
			Version: version,
			Desc:    t.Task.Message,
		},
	}

	tpl, err := template.ParseFS(templates, "templates/microsoft-teams.tmpl")

	if err != nil {
		t.Log("Can't parse microsoft teams alert template!")
		panic(err)
	}

	if err := tpl.Execute(body, alert); err != nil {
		t.Log("Can't generate microsoft teams alert template!")
		panic(err)
	}

	if body.Len() == 0 {
		t.Log("Buffer for microsoft teams alert is empty")
		return
	}

	t.Log("Attempting to send microsoft teams alert")

	resp, err := http.Post(
		util.Config.MicrosoftTeamsUrl,
		"application/json",
		body,
	)

	if err != nil {
		t.Log("Can't send microsoft teams alert! Error: " + err.Error())
	} else if resp.StatusCode != 200 && resp.StatusCode != 202 {
		t.Log("Can't send microsoft teams alert! Response code: " + strconv.Itoa(resp.StatusCode))
	} else {
		t.Log("Sent successfully microsoft teams alert")
	}
	if resp != nil {
		defer resp.Body.Close() //nolint:errcheck
	}
}

func (t *TaskRunner) sendDingTalkAlert() {
	if !util.Config.DingTalkAlert || !t.alert {
		return
	}

	if t.Template.SuppressSuccessAlerts && t.Task.Status == task_logger.TaskSuccessStatus {
		return
	}

	body := bytes.NewBufferString("")
	author, version := t.alertInfos()

	alert := Alert{
		Name:   t.Template.Name,
		Author: author,
		Color:  t.alertColor("dingtalk"),
		Task: alertTask{
			ID:      strconv.Itoa(t.Task.ID),
			URL:     t.taskLink(),
			Result:  t.Task.Status.Format(),
			Version: version,
			Desc:    t.Task.Message,
		},
	}

	tpl, err := template.ParseFS(templates, "templates/dingtalk.tmpl")

	if err != nil {
		t.Log("Can't parse dingtalk alert template!")
		panic(err)
	}

	if err := tpl.Execute(body, alert); err != nil {
		t.Log("Can't generate dingtalk alert template!")
		panic(err)
	}

	if body.Len() == 0 {
		t.Log("Buffer for dingtalk alert is empty")
		return
	}

	t.Log("Attempting to send dingtalk alert")

	resp, err := http.Post(
		util.Config.DingTalkUrl,
		"application/json",
		body,
	)

	if err != nil {
		t.Log("Can't send dingtalk alert! Error: " + err.Error())
	} else if resp.StatusCode != 200 {
		t.Log("Can't send dingtalk alert! Response code: " + strconv.Itoa(resp.StatusCode))
	} else {
		t.Log("Sent successfully dingtalk alert")
	}

	if resp != nil {
		defer resp.Body.Close() //nolint:errcheck
	}
}

func (t *TaskRunner) sendGotifyAlert() {
	if !util.Config.GotifyAlert || !t.alert {
		return
	}

	if t.Template.SuppressSuccessAlerts && t.Task.Status == task_logger.TaskSuccessStatus {
		return
	}

	body := bytes.NewBufferString("")
	author, version := t.alertInfos()

	alert := Alert{
		Name:   t.Template.Name,
		Author: author,
		Color:  t.alertColor("gotify"),
		Task: alertTask{
			ID:      strconv.Itoa(t.Task.ID),
			URL:     t.taskLink(),
			Result:  t.Task.Status.Format(),
			Version: version,
			Desc:    t.Task.Message,
		},
	}

	tpl, err := template.ParseFS(templates, "templates/gotify.tmpl")

	if err != nil {
		t.Log("Can't parse gotify alert template!")
		panic(err)
	}

	if err := tpl.Execute(body, alert); err != nil {
		t.Log("Can't generate gotify alert template!")
		panic(err)
	}

	if body.Len() == 0 {
		t.Log("Buffer for gotify alert is empty")
		return
	}

	t.Log("Attempting to send gotify alert")

	resp, err := http.Post(
		fmt.Sprintf(
			"%s/message?token=%s",
			util.Config.GotifyUrl,
			util.Config.GotifyToken),
		"application/json",
		body,
	)

	if err != nil {
		t.Log("Can't send gotify alert! Error: " + err.Error())
	} else if resp.StatusCode != 200 {
		t.Log("Can't send gotify alert! Response code: " + strconv.Itoa(resp.StatusCode))
	} else {
		t.Log("Sent successfully gotify alert")
	}

	if resp != nil {
		defer resp.Body.Close() //nolint:errcheck
	}
}

func (t *TaskRunner) alertInfos() (string, string) {
	version := ""

	if t.Task.Version != nil {
		version = *t.Task.Version
	} else if t.Template.Type != db.TemplateTask {
		v := t.Task.GetIncomingVersion(t.pool.store)
		if v != nil {
			version = "build " + *v
		} else {
			version = ""
		}
	} else {
		version = ""
	}

	author := "—"

	if t.Task.UserID != nil {
		user, err := t.pool.store.GetUser(*t.Task.UserID)

		if err != nil {
			panic(err)
		}

		author = user.Name
	}

	return author, version
}

func (t *TaskRunner) alertColor(kind string) string {
	switch kind {
	case "slack":
		switch t.Task.Status {
		case task_logger.TaskSuccessStatus:
			return "#2EFF2E"
		case task_logger.TaskFailStatus:
			return "#E50000"
		case task_logger.TaskRunningStatus:
			return "#333CFF"
		case task_logger.TaskWaitingStatus:
			return "#FFFC33"
		case task_logger.TaskStoppingStatus:
			return "#BEBEBE"
		case task_logger.TaskStoppedStatus:
			return "#5B5B5B"
		}
	case "rocketchat":
		switch t.Task.Status {
		case task_logger.TaskSuccessStatus:
			return "#00EE00"
		case task_logger.TaskFailStatus:
			return "#EE0000"
		case task_logger.TaskRunningStatus:
			return "#333CFF"
		case task_logger.TaskWaitingStatus:
			return "#FFFC33"
		case task_logger.TaskStoppingStatus:
			return "#BEBEBE"
		case task_logger.TaskStoppedStatus:
			return "#5B5B5B"
		}
	}

	return ""
}

func (t *TaskRunner) taskLink() string {
	return fmt.Sprintf(
		"%s/project/%d/templates/%d?t=%d",
		util.Config.WebHost,
		t.Template.ProjectID,
		t.Template.ID,
		t.Task.ID,
	)
}
