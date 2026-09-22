package infrastructure

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"github.com/skiix-backend/internal/domain"
	"google.golang.org/api/option"
)

type firebaseServiceImpl struct {
	app        *firebase.App
	msgClient  *messaging.Client
	authClient *auth.Client
}

func NewFirebaseService() domain.FirebaseService {
	return &firebaseServiceImpl{}
}

func (fs *firebaseServiceImpl) Init(credentialsFilePath string) error {
	opt := option.WithCredentialsFile(credentialsFilePath)

	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return fmt.Errorf("error initializing app: %v", err)
	}
	fs.app = app

	msgClient, err := app.Messaging(ctx)
	if err != nil {
		return fmt.Errorf("error getting Messaging client: %v", err)
	}
	fs.msgClient = msgClient

	authClient, err := app.Auth(ctx)
	if err != nil {
		return fmt.Errorf("error getting Auth client: %v", err)
	}
	fs.authClient = authClient

	log.Println("Firebase Admin SDK initialized successfully")
	return nil
}

func (fs *firebaseServiceImpl) SendPushNotification(tokens []string, title string, body string, data map[string]string) error {
	if fs.msgClient == nil {
		return fmt.Errorf("firebase messaging client is not initialized")
	}

	if len(tokens) == 0 {
		return nil
	}

	ctx := context.Background()

	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	response, err := fs.msgClient.SendMulticast(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send multicast message: %v", err)
	}

	if response.FailureCount > 0 {
		log.Printf("Firebase Multicast Sent with failures. Success: %d, Failure: %d", response.SuccessCount, response.FailureCount)
	}

	return nil
}

func (fs *firebaseServiceImpl) GenerateCustomToken(userID string) (string, error) {
	if fs.authClient == nil {
		return "", fmt.Errorf("firebase auth client is not initialized")
	}

	ctx := context.Background()
	token, err := fs.authClient.CustomToken(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("error minting custom token: %v", err)
	}

	return token, nil
}
