package kafka

import (
	"context"
	"encoding/json"
	"time"

	"csbs/backend/pkg/logger"

	kgo "github.com/segmentio/kafka-go"
)

// BookingEventHandler — callback, вызываемый консьюмером на каждое событие.
// Ошибка возвращается, чтобы консьюмер мог пометить сообщение как обработанное
// (commit) только при успехе. Сейчас kafka-go в режиме GroupID делает auto-commit,
// поэтому ошибка просто логируется.
type BookingEventHandler func(ctx context.Context, evt BookingEvent) error

// StartBookingConsumer запускает горутину, читающую TopicBookingEvents в группе groupID.
// Несколько групп = несколько независимых подписчиков одной и той же шины.
// При ctx.Done() консьюмер штатно закрывается.
func StartBookingConsumer(ctx context.Context, brokersCSV, groupID string, handler BookingEventHandler) {
	brokers := splitAndTrim(brokersCSV)
	if len(brokers) == 0 {
		logger.Warn.Printf("Kafka: KAFKA_BROKERS пуст — консьюмер %q не запущен", groupID)
		return
	}

	r := kgo.NewReader(kgo.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          TopicBookingEvents,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		StartOffset:    kgo.LastOffset, // новые сообщения с момента подписки
		CommitInterval: time.Second,
	})

	logger.Info.Printf("Kafka: консьюмер %q запущен (topic=%s)", groupID, TopicBookingEvents)

	go func() {
		defer r.Close()
		for {
			m, err := r.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					logger.Info.Printf("Kafka: консьюмер %q остановлен", groupID)
					return
				}
				logger.Error.Printf("Kafka [%s]: fetch failed: %v", groupID, err)
				time.Sleep(time.Second) // не спамим CPU при постоянной ошибке
				continue
			}

			var evt BookingEvent
			if err := json.Unmarshal(m.Value, &evt); err != nil {
				logger.Error.Printf("Kafka [%s]: invalid payload, skipping: %v", groupID, err)
				_ = r.CommitMessages(ctx, m) // битое сообщение не зацикливаем
				continue
			}

			if err := handler(ctx, evt); err != nil {
				logger.Error.Printf("Kafka [%s]: handler error for reservation=%d: %v", groupID, evt.ReservationID, err)
				// Всё равно коммитим, чтобы не зациклиться. В продакшене тут было бы DLQ.
			}

			if err := r.CommitMessages(ctx, m); err != nil {
				logger.Error.Printf("Kafka [%s]: commit failed: %v", groupID, err)
			}
		}
	}()
}
