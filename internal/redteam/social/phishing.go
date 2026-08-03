package social

import (
	"context"
	"fmt"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// SetupPhishDomain automates the deployment of GoPhish inside the sandbox for campaign management.
func SetupPhishDomain(ctx context.Context, domainName, targetSite string, sb *sandbox.Docker) (string, error) {
	// 1. Download and start GoPhish
	dlCmd := `if [ ! -d /opt/gophish ]; then
		wget -q https://github.com/gophish/gophish/releases/download/v0.12.1/gophish-v0.12.1-linux-64bit.zip -O /tmp/gophish.zip
		unzip -q /tmp/gophish.zip -d /opt/gophish
		chmod +x /opt/gophish/gophish
	fi
	cd /opt/gophish
	nohup ./gophish > /var/log/gophish.log 2>&1 &
	sleep 3`
	
	if _, err := sb.Execute(ctx, dlCmd); err != nil {
		return "", fmt.Errorf("failed to deploy GoPhish: %v", err)
	}

	// 2. Extract auto-generated admin password from logs
	extractCmd := `grep 'Please login with the username admin and the password' /var/log/gophish.log | awk '{print $NF}' | tr -d '"'`
	pass, err := sb.Execute(ctx, extractCmd)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve GoPhish credentials: %v", err)
	}

	return fmt.Sprintf("[+] GoPhish framework deployed in sandbox on port 3333.\n[!] Admin Credentials: admin / %s\n[!] Proceed to configure the %s campaign via the REST API or Web UI.", pass, targetSite), nil
}

// GeneratePhishEmail creates a highly tailored HTML spear-phishing template.
func GeneratePhishEmail(ctx context.Context, targetProfile string, sb *sandbox.Docker) (string, error) {
	// In a full implementation, we'd pass targetProfile to the LLM to generate the text.
	// For now, we generate a highly lethal generic IT template.
	template := `
<html>
<body style="font-family: Arial, sans-serif;">
	<h2 style="color: #d9534f;">URGENT: Mandatory IT Security Update</h2>
	<p>Hello,</p>
	<p>Our automated security systems have detected an anomalous login attempt on your account from an unrecognized device.</p>
	<p>As per company policy, you must verify your identity and update your session token immediately to prevent account suspension.</p>
	<p><a href="{{LURE_URL}}" style="background-color: #0275d8; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Account Now</a></p>
	<p>Failure to complete this within 24 hours will result in a temporary lock on your Active Directory profile.</p>
	<p>Thank you,<br>IT Security Team</p>
	<br>
	<p style="font-size: 10px; color: #777;">Reference Profile: %s</p>
</body>
</html>`
	
	htmlContent := fmt.Sprintf(template, targetProfile)
	
	writeCmd := fmt.Sprintf("cat << 'EOF' > /tmp/phish_template.html\n%s\nEOF", htmlContent)
	if _, err := sb.Execute(ctx, writeCmd); err != nil {
		return "", fmt.Errorf("failed to write template: %v", err)
	}

	return "[+] HTML Phishing template generated at /tmp/phish_template.html. Replace {{LURE_URL}} with the Evilginx2 lure.", nil
}

// SendPhish writes a Python script to send the email via an SMTP relay.
func SendPhish(ctx context.Context, targetEmail, template string, sb *sandbox.Docker) (string, error) {
	// Write a python script to send the email
	pyScript := fmt.Sprintf(`cat << 'EOF' > /tmp/send_phish.py
import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

# Configuration (Replace with actual SMTP relay credentials or use local postfix)
SMTP_SERVER = "127.0.0.1"
SMTP_PORT = 25
SENDER_EMAIL = "security@it-support-update.com"
TARGET_EMAIL = "%s"

msg = MIMEMultipart()
msg['From'] = SENDER_EMAIL
msg['To'] = TARGET_EMAIL
msg['Subject'] = "URGENT: Mandatory IT Security Update"

with open("/tmp/phish_template.html", "r") as f:
    html_content = f.read()

msg.attach(MIMEText(html_content, 'html'))

try:
    server = smtplib.SMTP(SMTP_SERVER, SMTP_PORT)
    # server.starttls()
    # server.login(SMTP_USER, SMTP_PASS)
    server.sendmail(SENDER_EMAIL, TARGET_EMAIL, msg.as_string())
    server.quit()
    print("Email sent successfully!")
except Exception as e:
    print(f"Failed to send email: {e}")
EOF
python3 /tmp/send_phish.py`, targetEmail)

	out, err := sb.Execute(ctx, pyScript)
	if err != nil {
		return "", fmt.Errorf("failed to execute phishing script: %w\nOutput: %s", err, out)
	}

	return fmt.Sprintf("[+] Phishing email sent to %s.\nOutput: %s", targetEmail, out), nil
}
