package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ResendEmailService struct {
	apiKey   string
	from     string
	fromName string
	baseURL  string
}

func NewResendEmailService(apiKey, from, fromName string) *ResendEmailService {
	return &ResendEmailService{
		apiKey:   apiKey,
		from:     from,
		fromName: fromName,
		baseURL:  "https://api.resend.com",
	}
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (s *ResendEmailService) sendEmail(ctx context.Context, to, subject, html string) error {
	reqBody := resendRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.from),
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	var respBody bytes.Buffer
	respBody.ReadFrom(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("email send failed with status %d: %s", resp.StatusCode, respBody.String())
	}

	return nil
}

func (s *ResendEmailService) SendVerificationEmail(ctx context.Context, to, name string, data map[string]interface{}) error {
	verificationURL := data["verification_url"].(string)
	expiresHours := data["expires_hours"].(int)

	subject := "Confirma tu registro en Vertercloud"
	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2 style="color: #333;">Hola %s,</h2>
			<p>Gracias por registrarte en Vertercloud.</p>
			<p>Para completar tu registro, por favor confirma tu correo electrónico:</p>

			<div style="background-color: #f5f5f5; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<p style="text-align: center;">
					<a href="%s" style="display: inline-block; padding: 12px 30px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; font-weight: bold;">Verificar Correo Electrónico</a>
				</p>
				<p style="text-align: center; color: #666; font-size: 14px; margin-top: 15px;">Este enlace expira en %d horas</p>
			</div>

			<p style="color: #666; font-size: 13px;">Si el botón no funciona, copia y pega este enlace en tu navegador:</p>
			<p style="word-break: break-all; color: #007bff; font-size: 12px;">%s</p>

			<p style="color: #999; font-size: 12px; margin-top: 30px;">Si no creaste una cuenta, puedes ignorar este correo con seguridad.</p>

			<p>Saludos,<br>El equipo de Vertercloud</p>
		</div>
	`, name, verificationURL, expiresHours, verificationURL)

	return s.sendEmail(ctx, to, subject, html)
}

func (s *ResendEmailService) SendPasswordReset(ctx context.Context, to, code, resetURL string) error {
	subject := "Recuperación de Contraseña"
	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2 style="color: #333;">Recuperación de Contraseña</h2>
			<p>Recibimos una solicitud para restablecer tu contraseña.</p>

			<div style="background-color: #f5f5f5; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h3 style="color: #007bff; margin-top: 0;">Opción 1: Usa este código de verificación</h3>
				<p style="font-size: 32px; font-weight: bold; letter-spacing: 5px; color: #007bff; text-align: center; margin: 15px 0;">%s</p>
				<p style="text-align: center; color: #666; font-size: 14px;">Este código expira en 15 minutos</p>
			</div>

			<div style="background-color: #f5f5f5; padding: 20px; border-radius: 5px; margin: 20px 0;">
				<h3 style="color: #28a745; margin-top: 0;">Opción 2: Haz clic en este enlace</h3>
				<p style="text-align: center;">
					<a href="%s" style="display: inline-block; padding: 12px 30px; background-color: #28a745; color: white; text-decoration: none; border-radius: 5px; font-weight: bold;">Restablecer Contraseña</a>
				</p>
				<p style="text-align: center; color: #666; font-size: 14px;">Este enlace expira en 15 minutos</p>
			</div>

			<p style="color: #999; font-size: 12px; margin-top: 30px;">Si no solicitaste esto, puedes ignorar este correo con seguridad.</p>
		</div>
	`, code, resetURL)

	return s.sendEmail(ctx, to, subject, html)
}

func (s *ResendEmailService) SendWelcome(ctx context.Context, to, name string) error {
	subject := "Welcome to Vertercloud!"
	html := fmt.Sprintf(`
		<h2>Welcome to Vertercloud, %s!</h2>
		<p>Your account has been successfully created.</p>
		<p>You can now log in and start using our services.</p>
	`, name)

	return s.sendEmail(ctx, to, subject, html)
}

func (s *ResendEmailService) Send2FAEnabled(ctx context.Context, to string) error {
	subject := "Two-Factor Authentication Enabled"
	html := `
		<h2>2FA Enabled</h2>
		<p>Two-factor authentication has been enabled on your account.</p>
		<p>If you didn't enable this, please contact support immediately.</p>
	`

	return s.sendEmail(ctx, to, subject, html)
}

func (s *ResendEmailService) SendSecurityAlert(ctx context.Context, to, subject, message string) error {
	html := fmt.Sprintf(`
		<h2>Security Alert</h2>
		<p>%s</p>
		<p>If this wasn't you, please secure your account immediately.</p>
	`, message)

	return s.sendEmail(ctx, to, subject, html)
}

func (s *ResendEmailService) SendPasswordExpiryWarning(ctx context.Context, to, name string, daysRemaining int) error {
	subject := "Tu contraseña expirará pronto"
	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2 style="color: #333;">Hola %s,</h2>
			<p>Te informamos que tu contraseña de Vertercloud expirará en <strong>%d días</strong> por políticas de seguridad.</p>
			<p>Para evitar interrupciones en tu acceso, te recomendamos cambiarla lo antes posible.</p>

			<div style="background-color: #fff3cd; padding: 20px; border-radius: 5px; border: 1px solid #ffeeba; margin: 20px 0;">
				<p style="margin: 0; color: #856404;"><strong>Importante:</strong> Una vez expirada, no podrás iniciar sesión hasta que restablezcas tu contraseña.</p>
			</div>

			<p>Puedes cambiar tu contraseña desde tu perfil de usuario o usando la opción de "Olvidé mi contraseña" en el login.</p>

			<p style="color: #999; font-size: 12px; margin-top: 30px;">Si ya cambiaste tu contraseña recientemente, puedes ignorar este correo.</p>

			<p>Saludos,<br>El equipo de Vertercloud</p>
		</div>
	`, name, daysRemaining)

	return s.sendEmail(ctx, to, subject, html)
}
