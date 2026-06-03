package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"csbs/backend/pkg/logger"

	kgo "github.com/segmentio/kafka-go"
)

// Producer — тонкая обёртка над kafka-go.Writer для публикации событий брони.
// Если brokers пустой — продюсер работает в no-op режиме: вызовы Publish
// просто пишут предупреждение в лог и не валят запрос.
type Producer struct {
	writer *kgo.Writer
}

// NewProducer создаёт продюсера. brokersCSV — список адресов через запятую (kafka:9092,...).
// Возвращает nil, если brokersCSV пуст (Kafka выключена в конфиге).
func NewProducer(brokersCSV string) *Producer {
	brokers := splitAndTrim(brokersCSV)
	if len(brokers) == 0 {
		logger.Warn.Println("Kafka: KAFKA_BROKERS пуст — продюсер не запущен (события публиковаться не будут)")
		return nil
	}

	w := &kgo.Writer{
		Addr:         kgo.TCP(brokers...),
		Balancer:     &kgo.Hash{}, // ключ → один и тот же раздел, сохраняем порядок по ReservationID
		BatchTimeout: 100 * time.Millisecond,
		RequiredAcks: kgo.RequireOne,
		Async:        false, // sync write, но мы вызываем из горутины — см. PublishAsync
		Compression:  kgo.Snappy,
	}
	logger.Info.Printf("Kafka: продюсер инициализирован, брокеры=%v", brokers)
	return &Producer{writer: w}
}

// PublishBookingEvent отправляет событие в TopicBookingEvents.
// Ключ = ReservationID, чтобы все события одной брони шли в один раздел и читались по порядку.
func (p *Producer) PublishBookingEvent(ctx context.Context, evt BookingEvent) error {
	if p == nil || p.writer == nil {
		return nil // no-op режим
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now()
	}
	body, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	msg := kgo.Message{
		Topic: TopicBookingEvents,
		Key:   []byte(strconv.FormatUint(uint64(evt.ReservationID), 10)),
		Value: body,
		Headers: []kgo.Header{
			{Key: "event-type", Value: []byte(evt.Type)},
		},
	}
	return p.writer.WriteMessages(ctx, msg)
}

// PublishAsync — fire-and-forget обёртка. Логирует ошибку, но не возвращает её,
// чтобы упавшая Kafka не ломала бизнес-операцию (создание брони).
func (p *Producer) PublishAsync(evt BookingEvent) {
	if p == nil || p.writer == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.PublishBookingEvent(ctx, evt); err != nil {
			logger.Error.Printf("Kafka: publish %s failed: %v", evt.Type, err)
			return
		}
		logger.Info.Printf("Kafka: published %s reservation=%d", evt.Type, evt.ReservationID)
	}()
}

func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

func splitAndTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
