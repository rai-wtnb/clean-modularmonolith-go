package eventhandlers

import (
	"fmt"
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/modules/shared/idempotent"
)

// NotificationSender sends notifications via external services.
// It embeds idempotent.Base so each outbound call is deduplicated on retry.
type NotificationSender struct {
	idempotent.Base
	logger *slog.Logger
}

func NewNotificationSender(logger *slog.Logger) *NotificationSender {
	return &NotificationSender{logger: logger}
}

func (s *NotificationSender) SendOrderConfirmation(orderID string) error {
	return s.Once(fmt.Sprintf("send-confirmation:%s", orderID), func() error {
		s.logger.Info("sending email to user", slog.String("order_id", orderID), slog.String("action", "order_confirmation"))
		return nil
	})
}
