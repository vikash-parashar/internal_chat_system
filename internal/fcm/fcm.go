package fcm

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var client *messaging.Client

func Init(pathToServiceAccount string) error {

	
	opt := option.WithCredentialsFile(pathToServiceAccount)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return err
	}

	client, err = app.Messaging(context.Background())
	if err != nil {
		return err
	}

	log.Println("🚀 FCM initialized")
	return nil
}

func SendPush(deviceToken, title, body string) error {
	if client == nil {
		return nil // Or panic/log
	}

	msg := &messaging.Message{
		Token: deviceToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}

	_, err := client.Send(context.Background(), msg)
	if err != nil {
		log.Printf("❌ FCM send failed: %v", err)
		return err
	}

	log.Printf("📲 Push sent to device token %s", deviceToken)
	return nil
}

// Step 1: Firebase Setup
// Go to Firebase Console

// Create or select your project.

// Go to Project Settings → Service accounts

// Click Generate new private key → Download the firebase-adminsdk-xxxxx.json file.
