package notifications

import (
	"context"
	"log"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

var client *messaging.Client

// Init initializes Firebase app with serviceAccountKey.json
func Init(serviceAccountPath string) error {
	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Printf("❌ Firebase init failed: %v", err)
		return err
	}

	client, err = app.Messaging(context.Background())
	if err != nil {
		log.Printf("❌ Failed to get FCM client: %v", err)
		return err
	}

	log.Println("✅ Firebase FCM initialized")
	return nil
}

// SendPush sends a push notification to the given FCM token
func SendPush(token, title, body string) error {
	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}

	_, err := client.Send(context.Background(), msg)
	if err != nil {
		log.Printf("❌ Failed to send push: %v", err)
		return err
	}

	log.Printf("📲 Push sent to %s", token)
	return nil
}
