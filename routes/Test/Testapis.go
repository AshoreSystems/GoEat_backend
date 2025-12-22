package test

import (
	"GoEatsapi/firebase"
	"GoEatsapi/utils"
	"net/http"
)

func TestSendNotification(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Method not allowed", nil)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}

	deviceToken := r.FormValue("device_token")
	if deviceToken == "" {
		utils.JSON(w, 400, false, "device_token is required", nil)
		return
	}

	title := "Test Notification 🔔"
	body := "This is a test push notification from GoEats."

	err = firebase.SendPushNotification(
		firebase.App,
		deviceToken,
		title,
		body,
	)

	if err != nil {
		utils.JSON(w, 500, false, "Failed to send notification", err.Error())
		return
	}

	utils.JSON(w, 200, true, "Notification sent successfully", nil)
}
