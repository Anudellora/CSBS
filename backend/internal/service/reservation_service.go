package service

import (
	"csbs/backend/internal/models"
	"csbs/backend/internal/repository"
	"csbs/backend/pkg/kafka"
	"csbs/backend/pkg/logger"
	"errors"
	"time"
)

type ReservationService interface {
	CreateReservation(userID, workspaceID, tariffID uint, startTime, endTime time.Time) (*models.Reservation, error)
	GetUserReservations(userID uint) ([]models.Reservation, error)
	GetAllReservations() ([]models.Reservation, error)
	GetReservationByID(id uint) (*models.Reservation, error)
	UpdateReservation(id uint, workspaceID, tariffID uint, startTime, endTime time.Time, status string, actorUserID uint) (*models.Reservation, error)
	DeleteReservation(id, actorUserID uint) error
	GetUnavailableWorkspaceIDs(startTime, endTime time.Time) ([]uint, error)
}
type reservationServiceImpl struct {
	repo      repository.ReservationRepository
	auditRepo repository.AuditRepository
	producer  *kafka.Producer // nil → события не публикуются, остальное работает
}

func NewReservationService(repo repository.ReservationRepository, auditRepo repository.AuditRepository, producer *kafka.Producer) ReservationService {
	return &reservationServiceImpl{repo: repo, auditRepo: auditRepo, producer: producer}
}

// loadBookingEvent дотягивает связанные сущности и собирает денормализованный
// payload для Kafka. Если бронь по какой-то причине недоступна — публикуется
// минимальный набор полей.
func (s *reservationServiceImpl) publishBookingEvent(eventType string, id uint) {
	if s.producer == nil {
		return
	}
	r, err := s.repo.GetByID(id)
	if err != nil || r == nil {
		logger.Warn.Printf("Kafka: cannot enrich %s for reservation=%d: %v", eventType, id, err)
		return
	}
	s.producer.PublishAsync(kafka.BookingEvent{
		Type:          eventType,
		ReservationID: r.ID,
		UserID:        r.UserID,
		UserEmail:     r.User.Email,
		UserName:      r.User.FullName,
		WorkspaceID:   r.WorkspaceID,
		WorkspaceName: r.Workspace.NameOrNumber,
		LocationName:  r.Workspace.Location.Name,
		StartTime:     r.StartTime,
		EndTime:       r.EndTime,
		Status:        r.Status,
	})
}
func (s *reservationServiceImpl) CreateReservation(userID, workspaceID, tariffID uint, startTime, endTime time.Time) (*models.Reservation, error) {
	logger.Info.Printf("Service: Creating reservation for UserID %d, WorkspaceID %d", userID, workspaceID)
	// Проверяем, не занято ли место на это время
	conflict, err := s.repo.HasConflict(workspaceID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	if conflict {
		logger.Warn.Printf("Service: Reservation conflict for WorkspaceID %d from %s to %s", workspaceID, startTime, endTime)
		return nil, errors.New("это место уже забронировано на выбранное время")
	}
	reservation := &models.Reservation{
		UserID:      userID,
		WorkspaceID: workspaceID,
		TariffID:    tariffID,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      "подтверждено",
	}
	err = s.repo.Create(reservation)
	if err == nil {
		logger.Info.Printf("Service: Successfully created reservation ID %d", reservation.ID)
		s.auditRepo.Create(&models.AuditLog{
			UserID:     userID,
			Action:     "Create Reservation",
			EntityType: "Reservation",
			EntityID:   reservation.ID,
			Timestamp:  time.Now(),
		})
		s.publishBookingEvent(kafka.EventBookingCreated, reservation.ID)
	} else {
		logger.Error.Printf("Service: Error creating reservation: %v", err)
	}
	return reservation, err
}
func (s *reservationServiceImpl) GetUserReservations(userID uint) ([]models.Reservation, error) {
	logger.Info.Printf("Service: Requesting reservations for UserID: %d", userID)
	return s.repo.GetByUserID(userID)
}

func (s *reservationServiceImpl) GetAllReservations() ([]models.Reservation, error) {
	logger.Info.Println("Service: Requesting all reservations (Admin)")
	return s.repo.GetAll()
}

func (s *reservationServiceImpl) GetReservationByID(id uint) (*models.Reservation, error) {
	return s.repo.GetByID(id)
}

func (s *reservationServiceImpl) UpdateReservation(id uint, workspaceID, tariffID uint, startTime, endTime time.Time, status string, actorUserID uint) (*models.Reservation, error) {
	logger.Info.Printf("Service: Updating reservation ID %d", id)
	reservation, err := s.repo.GetByID(id)
	if err != nil {
		logger.Error.Printf("Service: Reservation %d not found: %v", id, err)
		return nil, errors.New("бронирование не найдено")
	}

	if status != "отменено" {
		conflict, err := s.repo.HasConflictExcluding(workspaceID, startTime, endTime, id)
		if err != nil {
			return nil, err
		}
		if conflict {
			return nil, errors.New("это место уже забронировано на выбранное время")
		}
	}

	reservation.WorkspaceID = workspaceID
	reservation.TariffID = tariffID
	reservation.StartTime = startTime
	reservation.EndTime = endTime
	if status != "" {
		reservation.Status = status
	}

	if err := s.repo.Update(reservation); err != nil {
		logger.Error.Printf("Service: Error updating reservation %d: %v", id, err)
		return nil, err
	}

	s.auditRepo.Create(&models.AuditLog{
		UserID:     actorUserID,
		Action:     "Update Reservation",
		EntityType: "Reservation",
		EntityID:   reservation.ID,
		Timestamp:  time.Now(),
	})

	eventType := kafka.EventBookingUpdated
	if status == "отменено" {
		eventType = kafka.EventBookingCancelled
	}
	s.publishBookingEvent(eventType, reservation.ID)

	return s.repo.GetByID(id)
}

func (s *reservationServiceImpl) DeleteReservation(id, actorUserID uint) error {
	logger.Info.Printf("Service: Deleting reservation ID %d", id)
	// Снимаем снапшот до удаления, чтобы консьюмеры получили нормальный payload.
	snapshot, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("бронирование не найдено")
	}
	if err := s.repo.Delete(id); err != nil {
		logger.Error.Printf("Service: Error deleting reservation %d: %v", id, err)
		return err
	}
	s.auditRepo.Create(&models.AuditLog{
		UserID:     actorUserID,
		Action:     "Delete Reservation",
		EntityType: "Reservation",
		EntityID:   id,
		Timestamp:  time.Now(),
	})
	if s.producer != nil && snapshot != nil {
		s.producer.PublishAsync(kafka.BookingEvent{
			Type:          kafka.EventBookingCancelled,
			ReservationID: snapshot.ID,
			UserID:        snapshot.UserID,
			UserEmail:     snapshot.User.Email,
			UserName:      snapshot.User.FullName,
			WorkspaceID:   snapshot.WorkspaceID,
			WorkspaceName: snapshot.Workspace.NameOrNumber,
			LocationName:  snapshot.Workspace.Location.Name,
			StartTime:     snapshot.StartTime,
			EndTime:       snapshot.EndTime,
			Status:        "удалено",
		})
	}
	return nil
}

func (s *reservationServiceImpl) GetUnavailableWorkspaceIDs(startTime, endTime time.Time) ([]uint, error) {
	return s.repo.GetUnavailableWorkspaceIDs(startTime, endTime)
}
