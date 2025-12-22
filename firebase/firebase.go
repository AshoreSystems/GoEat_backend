package firebase

import (
	"GoEatsapi/utils"
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var App *firebase.App

func InitFirebase() {
	opt := option.WithCredentialsFile("config/goeats-dev-firebase-adminsdk-fbsvc-ae62dd3c54.json")

	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		utils.InfoLog.Println("firebase init failed:", err)
	}

	App = app
	utils.InfoLog.Println("Firebase initialized successfully")
}

func SendPushNotification(app *firebase.App, deviceToken string, title string, body string) error {
	ctx := context.Background()

	client, err := app.Messaging(ctx)
	if err != nil {
		return err
	}

	message := &messaging.Message{
		Token: deviceToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: map[string]string{
			"type": "order_update",
		},
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		return err
	}

	log.Println("Successfully sent message:", response)
	return nil
}
