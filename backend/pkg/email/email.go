package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"

	"csbs/backend/pkg/logger"
)

type Sender struct {
	host     string
	port     string
	user     string
	password string
}

func NewSender(host, port, user, password string) *Sender {
	return &Sender{host: host, port: port, user: user, password: password}
}

func (s *Sender) Configured() bool {
	return s.host != "" && s.user != "" && s.password != ""
}

func (s *Sender) SendPasswordReset(to, resetURL string) error {
	logger.Info.Printf("SMTP: attempting to send password reset to %s via %s:%s", to, s.host, s.port)

	msg := buildMessage(to, s.user, resetURL)
	return s.send(to, msg, "password reset")
}

// BookingReminder содержит данные брони для письма-напоминания.
type BookingReminder struct {
	To            string
	UserName      string
	WorkspaceName string
	LocationName  string
	StartTime     time.Time
	EndTime       time.Time
	HoursBefore   int // 24 или 3 — для текста темы/тела
}

// SendBookingReminder шлёт письмо-напоминание о ближайшей броне.
func (s *Sender) SendBookingReminder(rem BookingReminder) error {
	logger.Info.Printf("SMTP: sending %dh booking reminder to %s", rem.HoursBefore, rem.To)
	msg := buildReminderMessage(s.user, rem)
	return s.send(rem.To, msg, fmt.Sprintf("%dh booking reminder", rem.HoursBefore))
}

// SendBookingConfirmation шлёт письмо «бронь подтверждена» сразу после создания.
// Вызывается из Kafka-консьюмера на событие booking.created.
func (s *Sender) SendBookingConfirmation(rem BookingReminder) error {
	logger.Info.Printf("SMTP: sending booking confirmation to %s", rem.To)
	msg := buildConfirmationMessage(s.user, rem)
	return s.send(rem.To, msg, "booking confirmation")
}

// send — общий путь отправки: TLS на 465, иначе STARTTLS.
func (s *Sender) send(to, msg, label string) error {
	var err error
	if s.port == "465" {
		err = s.sendTLS(to, msg)
	} else {
		err = s.sendSTARTTLS(to, msg)
	}
	if err != nil {
		logger.Error.Printf("SMTP: %s send failed (%s:%s) → %v", label, s.host, s.port, err)
		return err
	}
	logger.Info.Printf("SMTP: %s delivered to %s", label, to)
	return nil
}

// sendSTARTTLS — порт 587, соединение открытое, затем апгрейд до TLS
func (s *Sender) sendSTARTTLS(to, msg string) error {
	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	return smtp.SendMail(s.host+":"+s.port, auth, s.user, []string{to}, []byte(msg))
}

// sendTLS — порт 465, сразу TLS-соединение (SMTPS)
func (s *Sender) sendTLS(to, msg string) error {
	tlsCfg := &tls.Config{ServerName: s.host}

	conn, err := tls.Dial("tcp", s.host+":"+s.port, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err = client.Mail(s.user); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	return w.Close()
}

func buildConfirmationMessage(from string, rem BookingReminder) string {
	subject := "Бронь в коворкинге CSBS подтверждена"

	greeting := "Здравствуйте!"
	if rem.UserName != "" {
		greeting = fmt.Sprintf("Здравствуйте, %s!", rem.UserName)
	}

	location := rem.LocationName
	if location == "" {
		location = "—"
	}

	body := fmt.Sprintf(
		"%s\n\nВаша бронь успешно создана.\n\n"+
			"• Место: %s\n"+
			"• Локация: %s\n"+
			"• Дата: %s\n"+
			"• Время: %s — %s (МСК)\n\n"+
			"Мы пришлём напоминание за сутки и за 3 часа до начала. "+
			"QR-пропуск для входа доступен в личном кабинете.\n\n"+
			"С уважением,\nКоманда CSBS",
		greeting,
		rem.WorkspaceName,
		location,
		rem.StartTime.Format("02.01.2006"),
		rem.StartTime.Format("15:04"),
		rem.EndTime.Format("15:04"),
	)

	return fmt.Sprintf(
		"To: %s\r\nFrom: CSBS <%s>\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		rem.To, from, subject, body,
	)
}

func buildReminderMessage(from string, rem BookingReminder) string {
	when := "через сутки"
	if rem.HoursBefore <= 3 {
		when = "через 3 часа"
	}
	subject := fmt.Sprintf("Напоминание: ваша бронь в коворкинге %s", when)

	greeting := "Здравствуйте!"
	if rem.UserName != "" {
		greeting = fmt.Sprintf("Здравствуйте, %s!", rem.UserName)
	}

	location := rem.LocationName
	if location == "" {
		location = "—"
	}

	body := fmt.Sprintf(
		"%s\n\nНапоминаем, что %s начинается ваша бронь в коворкинге CSBS.\n\n"+
			"• Место: %s\n"+
			"• Локация: %s\n"+
			"• Дата: %s\n"+
			"• Время: %s — %s (МСК)\n\n"+
			"Ждём вас! Если планы изменились, отмените бронь в личном кабинете.\n\n"+
			"С уважением,\nКоманда CSBS",
		greeting,
		when,
		rem.WorkspaceName,
		location,
		rem.StartTime.Format("02.01.2006"),
		rem.StartTime.Format("15:04"),
		rem.EndTime.Format("15:04"),
	)

	return fmt.Sprintf(
		"To: %s\r\nFrom: CSBS <%s>\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		rem.To, from, subject, body,
	)
}

func buildMessage(to, from, resetURL string) string {
	subject := "Восстановление пароля CSBS"
	body := fmt.Sprintf(
		"Здравствуйте!\n\nВы запросили восстановление пароля для вашего аккаунта CSBS.\n\nПерейдите по ссылке для создания нового пароля:\n%s\n\nСсылка действительна в течение 1 часа.\nЕсли вы не запрашивали восстановление пароля — просто проигнорируйте это письмо.\n\nС уважением,\nКоманда CSBS",
		resetURL,
	)
	return fmt.Sprintf(
		"To: %s\r\nFrom: CSBS <%s>\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		to, from, subject, body,
	)
}
