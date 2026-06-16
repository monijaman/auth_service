package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	gomail "gopkg.in/gomail.v2"
)

type Service struct {
	dialer *gomail.Dialer
	from   string
}

func New(host string, port int, username, password, from string) *Service {
	d := gomail.NewDialer(host, port, username, password)
	return &Service{dialer: d, from: from}
}

func (s *Service) SendVerificationEmail(_ context.Context, toEmail, code string) error {
	body, err := render(verificationTpl, map[string]string{"Code": code})
	if err != nil {
		return err
	}
	return s.send(toEmail, "Verify your email address", body)
}

func (s *Service) SendPasswordResetEmail(_ context.Context, toEmail, code string) error {
	body, err := render(passwordResetTpl, map[string]string{"Code": code})
	if err != nil {
		return err
	}
	return s.send(toEmail, "Reset your password", body)
}

func (s *Service) send(to, subject, html string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", html)
	return s.dialer.DialAndSend(m)
}

func render(tpl string, data map[string]string) (string, error) {
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("email template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const verificationTpl = `<!DOCTYPE html>
<html>
<body>
  <h2>Email Verification</h2>
  <p>Your verification code is: <strong>{{.Code}}</strong></p>
  <p>This code expires in 15 minutes.</p>
</body>
</html>`

const passwordResetTpl = `<!DOCTYPE html>
<html>
<body>
  <h2>Password Reset</h2>
  <p>Your password reset code is: <strong>{{.Code}}</strong></p>
  <p>This code expires in 15 minutes.</p>
</body>
</html>`
