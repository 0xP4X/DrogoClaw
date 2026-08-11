package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// Provider enumerates the CSPM providers Prowler supports. DrogonClaw's
// existing aws.go covers only AWS IAM enumeration; this wrapper extends the
// cloud package to multi-cloud posture management by orchestrating Prowler
// (wrapped, never forked) so we absorb its 1000+ checks without duplicating them.
type Provider string

const (
	ProviderAWS        Provider = "aws"
	ProviderAzure      Provider = "azure"
	ProviderGCP        Provider = "gcp"
	ProviderKubernetes Provider = "kubernetes"
)

// Options controls a single Prowler scan.
type Options struct {
	// Account / subscription / project id (provider-specific). Optional; when
	// empty Prowler uses the credentials already present in the sandbox.
	Account string
	// Region narrows the scan (AWS/Azure/GCP). Optional.
	Region string
	// OutDir is where Prowler writes its JSON inside the sandbox.
	OutDir string
	// Env carries provider credentials as environment variables that are
	// exported into the sandbox before the scan (e.g. AWS_ACCESS_KEY_ID, …).
	// They are never persisted to disk by DrogonClaw.
	Env map[string]string
}

// Finding is one Prowler result, mapped to DrogonClaw's finding shape.
type Finding struct {
	Provider string
	Account  string
	CheckID  string
	Status   string // FAIL | PASS | INFO | MANUAL
	Severity string // critical | high | medium | low | informational
	Service  string
	Region   string
	Resource string
	Message  string
}

// RunProwler executes Prowler against a provider inside the sandbox and returns
// parsed FAIL findings. Prowler must be installed in the sandbox image
// (pip install prowler); this function orchestrates it, it does not reimplement
// any checks.
func RunProwler(ctx context.Context, sb *sandbox.Docker, provider Provider, opts Options) ([]Finding, error) {
	if opts.OutDir == "" {
		opts.OutDir = "/workspace/prowler_output"
	}

	var b strings.Builder
	for k, v := range opts.Env {
		b.WriteString(fmt.Sprintf("export %s=%q && ", k, v))
	}
	scanCmd := fmt.Sprintf(
		"%smkdir -p %s && prowler %s scan --status FAIL --output-format json -F %s/results.json 2>&1; echo __PROWLER_DONE__",
		b.String(), opts.OutDir, string(provider), opts.OutDir,
	)
	if opts.Account != "" {
		scanCmd = strings.Replace(scanCmd, "prowler "+string(provider)+" scan",
			fmt.Sprintf("prowler %s scan --account %s", string(provider), opts.Account), 1)
	}
	if opts.Region != "" {
		scanCmd = strings.Replace(scanCmd, "prowler "+string(provider)+" scan",
			fmt.Sprintf("prowler %s scan --region %s", string(provider), opts.Region), 1)
	}

	if _, err := sb.Execute(ctx, scanCmd); err != nil {
		return nil, fmt.Errorf("prowler run failed: %w", err)
	}

	raw, err := sb.Execute(ctx, "cat "+opts.OutDir+"/results.json")
	if err != nil {
		return nil, fmt.Errorf("reading prowler results: %w", err)
	}
	return parseProwlerJSON(raw, string(provider), opts.Account), nil
}

// parseProwlerJSON tolerates schema drift across Prowler versions by reading
// findings as generic maps and extracting the common fields.
func parseProwlerJSON(raw, provider, account string) []Finding {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil
	}
	var out []Finding
	for _, row := range rows {
		status := str(row["Status"])
		if status == "" {
			status = str(row["StatusReason"])
		}
		if strings.EqualFold(status, "PASS") || strings.EqualFold(status, "INFO") {
			continue // report only FAIL/MANUAL by default
		}
		f := Finding{
			Provider: provider,
			Account:  orDefault(account, str(row["AccountId"]), str(row["SubscriptionId"]), str(row["ProjectId"])),
			CheckID:  orDefault(str(row["CheckID"]), str(row["CheckId"]), str(row["Check"])),
			Status:   status,
			Severity: str(row["Severity"]),
			Service:  str(row["ServiceName"]),
			Region:   str(row["Region"]),
			Resource: orDefault(str(row["ResourceId"]), str(row["ResourceArn"]), str(row["Resource"])),
		}
		if info, ok := row["FindingInfo"].(map[string]any); ok {
			f.Message = str(info["Description"])
		}
		if f.Message == "" {
			f.Message = orDefault(str(row["Message"]), str(row["Description"]))
		}
		out = append(out, f)
	}
	return out
}

func str(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		return fmt.Sprintf("%.0f", x)
	default:
		return ""
	}
}

func orDefault(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
