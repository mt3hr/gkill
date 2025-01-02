package gkill_notification

import (
	"encoding/json"
	"fmt"

	"github.com/SherClockHolmes/webpush-go"
)

type GkillNotificator struct {
	vapidPublicKey  string
	vapidPrivateKey string
}

func NewGkillNotificator(publicKey string, privateKey string) (*GkillNotificator, error) {
	return &GkillNotificator{
		vapidPublicKey:  publicKey,
		vapidPrivateKey: privateKey,
	}, nil
}

func (g *GkillNotificator) SendNotification(subscriptions []string, content string) error {
	for _, subscription := range subscriptions {
		s := &webpush.Subscription{}
		json.Unmarshal([]byte(subscription), s)
		resp, err := webpush.SendNotification([]byte(content), s, &webpush.Options{
			Subscriber:      "example@example.com",
			VAPIDPublicKey:  g.vapidPublicKey,
			VAPIDPrivateKey: g.vapidPrivateKey,
			TTL:             30,
		})
		if err != nil {
			err = fmt.Errorf("error at send gkill notification: %w", err)
		}
		defer resp.Body.Close()
	}
}
