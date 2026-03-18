package eventhandlers

import (
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/modules/shared/idempotent"
)

// NotificationSender sends notifications via external services.
// It embeds idempotent.OutboundCache so each outbound call is deduplicated on retry.
type NotificationSender struct {
	*idempotent.OutboundCache
	logger *slog.Logger
}

func NewNotificationSender(logger *slog.Logger) (_ *NotificationSender, cleanup func()) {
	cache, cleanup := idempotent.NewOutboundCache()
	return &NotificationSender{
		OutboundCache: cache,
		logger:        logger,
	}, cleanup
}

func (s *NotificationSender) SendOrderConfirmation(orderID string) error {
	return s.Once("send-confirmation", orderID, func() error {
		s.logger.Info("sending email to user", slog.String("order_id", orderID), slog.String("action", "order_confirmation"))
		return nil
	})
}

// SendOrderShipped sends a shipment notification and returns the external
// message ID. On Spanner retry the cached message ID is returned without
// re-sending.
func (s *NotificationSender) SendOrderShipped(orderID, trackingNumber string) (string, error) {
	return idempotent.OnceResult(s.OutboundCache, "send-shipped", orderID, func() (string, error) {
		s.logger.Info("sending shipment notification", slog.String("order_id", orderID), slog.String("tracking_number", trackingNumber))
		// In production this would call an external email/SMS API and return the provider's message ID.
		messageID := "msg-" + orderID + "-" + trackingNumber
		return messageID, nil
	})
}
