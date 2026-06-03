package service

import (
	"context"
	"fmt"

	"csbs/backend/internal/models"
	"csbs/backend/internal/repository"
	"csbs/backend/pkg/email"
	csbskafka "csbs/backend/pkg/kafka"
	"csbs/backend/pkg/logger"
)

// StartBookingEventConsumers поднимает двух независимых подписчиков на topic
// booking.events: один шлёт письма, второй пишет в audit_logs. Разные groupID
// гарантируют, что оба получат каждое сообщение.
//
// Безопасно вызывать с пустым brokersCSV — внутри будет no-op.
func StartBookingEventConsumers(
	ctx context.Context,
	brokersCSV string,
	mailer *email.Sender,
	auditRepo repository.AuditRepository,
) {
	csbskafka.StartBookingConsumer(ctx, brokersCSV, "csbs-notifications", notificationHandler(mailer))
	csbskafka.StartBookingConsumer(ctx, brokersCSV, "csbs-audit-mirror", auditMirrorHandler(auditRepo))
}

func notificationHandler(mailer *email.Sender) csbskafka.BookingEventHandler {
	return func(ctx context.Context, evt csbskafka.BookingEvent) error {
		if mailer == nil || !mailer.Configured() {
			return nil
		}
		if evt.UserEmail == "" {
			return nil
		}

		switch evt.Type {
		case csbskafka.EventBookingCreated:
			return mailer.SendBookingConfirmation(email.BookingReminder{
				To:            evt.UserEmail,
				UserName:      evt.UserName,
				WorkspaceName: evt.WorkspaceName,
				LocationName:  evt.LocationName,
				StartTime:     evt.StartTime,
				EndTime:       evt.EndTime,
			})
		case csbskafka.EventBookingCancelled:
			// Тут можно было бы отправить отдельное письмо об отмене.
			// Сейчас просто фиксируем в логе, чтобы видеть прохождение события.
			logger.Info.Printf("Notifications: booking cancelled, user=%s reservation=%d",
				evt.UserEmail, evt.ReservationID)
			return nil
		default:
			return nil
		}
	}
}

func auditMirrorHandler(auditRepo repository.AuditRepository) csbskafka.BookingEventHandler {
	return func(ctx context.Context, evt csbskafka.BookingEvent) error {
		// Зеркалим событие в audit_logs с пометкой, что источник — Kafka,
		// чтобы не путать с записями от прямого вызова из сервиса.
		return auditRepo.Create(&models.AuditLog{
			UserID:     evt.UserID,
			Action:     fmt.Sprintf("kafka:%s", evt.Type),
			EntityType: "Reservation",
			EntityID:   evt.ReservationID,
			Timestamp:  evt.OccurredAt,
		})
	}
}
