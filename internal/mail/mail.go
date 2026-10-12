package mail

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
)

const (
	smtpHost = "sandbox.smtp.mailtrap.io"
	smtpPort = "2525"
)

// GenerateOTP mirrors src/utils/mail.ts generateOTP().
func GenerateOTP() string {
	otp := ""
	for i := 0; i < 4; i++ {
		otp += fmt.Sprintf("%d", rand.Intn(10))
	}
	return otp
}

type Mailer struct {
	User string
	Pass string
}

func NewMailer(user, pass string) *Mailer {
	return &Mailer{User: user, Pass: pass}
}

// Send mirrors the nodemailer transporter.sendMail(...) calls in
// auth.controller.ts: same Mailtrap sandbox host/port, from address,
// subject, plain-text and HTML bodies.
func (m *Mailer) Send(to, subject, text, html string) error {
	from := `"Your App" <noreply@careNavigator.com>`
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n",
		from, to, subject, html,
	)

	auth := smtp.PlainAuth("", m.User, m.Pass, smtpHost)
	addr := smtpHost + ":" + smtpPort

	conn, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if ok, _ := conn.Extension("STARTTLS"); ok {
		if err := conn.StartTLS(&tls.Config{ServerName: smtpHost}); err != nil {
			return err
		}
	}

	if err := conn.Auth(auth); err != nil {
		return err
	}
	if err := conn.Mail(from); err != nil {
		return err
	}
	if err := conn.Rcpt(to); err != nil {
		return err
	}

	w, err := conn.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	return conn.Quit()
}
