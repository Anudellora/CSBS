package kafka

import "time"

// Названия топиков. Один источник правды для продюсеров и консьюмеров.
const (
	TopicBookingEvents = "booking.events"
)

// Типы событий брони, попадающие в TopicBookingEvents.
const (
	EventBookingCreated   = "booking.created"
	EventBookingUpdated   = "booking.updated"
	EventBookingCancelled = "booking.cancelled"
)

// BookingEvent — полезная нагрузка для событий жизненного цикла брони.
// Денормализована: содержит всё, что нужно консьюмеру (email, имя места и т.д.),
// чтобы не ходить за этим в БД на каждое сообщение.
type BookingEvent struct {
	Type          string    `json:"type"`
	ReservationID uint      `json:"reservation_id"`
	UserID        uint      `json:"user_id"`
	UserEmail     string    `json:"user_email,omitempty"`
	UserName      string    `json:"user_name,omitempty"`
	WorkspaceID   uint      `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name,omitempty"`
	LocationName  string    `json:"location_name,omitempty"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Status        string    `json:"status,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}
