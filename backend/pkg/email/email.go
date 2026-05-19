package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

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

	var err error
	if s.port == "465" {
		err = s.sendTLS(to, msg)
	} else {
		err = s.sendSTARTTLS(to, msg)
	}

	if err != nil {
		logger.Error.Printf("SMTP: send failed (%s:%s) → %v", s.host, s.port, err)
		return err
	}
	logger.Info.Printf("SMTP: password reset email delivered to %s", to)
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
