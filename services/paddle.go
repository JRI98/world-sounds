package services

import (
	"errors"
	"os"

	"github.com/PaddleHQ/paddle-go-sdk"
)

type PaddleService struct {
	WebhookVerifier *paddle.WebhookVerifier
}

func NewPaddleService() (*PaddleService, error) {
	webhookSecretKey, ok := os.LookupEnv("PADDLE_WEBHOOK_SECRET_KEY")
	if !ok {
		return nil, errors.New("PADDLE_WEBHOOK_SECRET_KEY environment variable not set")
	}

	webhookVerifier := paddle.NewWebhookVerifier(webhookSecretKey)

	return &PaddleService{
		WebhookVerifier: webhookVerifier,
	}, nil
}
