package cvss

import (
	"fmt"
	"math"
	"strings"
)

// ═══════════════════════════════════════════════════════════
// CVSS v3.1 Deterministic Scoring Engine
// Implements the FIRST CVSS v3.1 specification for calculating
// Base, Temporal, and Environmental scores.
// https://www.first.org/cvss/v3.1/specification-document
// ═══════════════════════════════════════════════════════════

// AttackVector (AV) values
type AttackVector string

const (
	AVNetwork         AttackVector = "N"
	AVAdjacent        AttackVector = "A"
	AVLocal           AttackVector = "L"
	AVPhysical        AttackVector = "P"
)

// AttackComplexity (AC) values
type AttackComplexity string

const (
	ACLow  AttackComplexity = "L"
	ACHigh AttackComplexity = "H"
)

// PrivilegesRequired (PR) values
type PrivilegesRequired string

const (
	PRNone PrivilegesRequired = "N"
	PRLow  PrivilegesRequired = "L"
	PRHigh PrivilegesRequired = "H"
)

// UserInteraction (UI) values
type UserInteraction string

const (
	UINone     UserInteraction = "N"
	UIRequired UserInteraction = "R"
)

// Scope (S) values
type Scope string

const (
	SUnchanged Scope = "U"
	SChanged   Scope = "C"
)

// Impact (C/I/A) values
type Impact string

const (
	ImpactHigh Impact = "H"
	ImpactLow  Impact = "L"
	ImpactNone Impact = "N"
)

// BaseMetrics contains all CVSS v3.1 base metric values.
type BaseMetrics struct {
	AttackVector          AttackVector       `json:"attackVector"`
	AttackComplexity      AttackComplexity   `json:"attackComplexity"`
	PrivilegesRequired    PrivilegesRequired `json:"privilegesRequired"`
	UserInteraction       UserInteraction    `json:"userInteraction"`
	Scope                 Scope              `json:"scope"`
	ConfidentialityImpact Impact             `json:"confidentialityImpact"`
	IntegrityImpact       Impact             `json:"integrityImpact"`
	AvailabilityImpact    Impact             `json:"availabilityImpact"`
}

// Severity level classification
type Severity string

const (
	SeverityNone     Severity = "None"
	SeverityLow      Severity = "Low"
	SeverityMedium   Severity = "Medium"
	SeverityHigh     Severity = "High"
	SeverityCritical Severity = "Critical"
)

// Result holds the computed CVSS score and metadata.
type Result struct {
	Score       float64  `json:"score"`       // 0.0–10.0
	Severity    Severity `json:"severity"`    // None, Low, Medium, High, Critical
	Vector      string   `json:"vector"`      // CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H
	Breakdown   string   `json:"breakdown"`   // human-readable breakdown
}

// CalculateBaseScore computes the CVSS v3.1 base score from the given metrics.
func CalculateBaseScore(m BaseMetrics) Result {
	// Step 1: Look up metric values
	avVal := attackVectorValue(m.AttackVector)
	acVal := attackComplexityValue(m.AttackComplexity)
	prVal := privReqValue(m.PrivilegesRequired, m.Scope)
	uiVal := userInteractionValue(m.UserInteraction)

	confVal := impactValue(m.ConfidentialityImpact)
	integVal := impactValue(m.IntegrityImpact)
	availVal := impactValue(m.AvailabilityImpact)

	// Step 2: Calculate Impact Sub Score (ISS)
	iss := 1.0 - ((1.0 - confVal) * (1.0 - integVal) * (1.0 - availVal))

	// Step 3: Calculate Impact
	var impact float64
	if m.Scope == SUnchanged {
		impact = 6.42 * iss
	} else {
		impact = 7.52*(iss-0.029) - 3.25*math.Pow(iss-0.02, 15)
	}

	// Step 4: Calculate Exploitability
	exploitability := 8.22 * avVal * acVal * prVal * uiVal

	// Step 5: Calculate Base Score
	var score float64
	if impact <= 0 {
		score = 0
	} else if m.Scope == SUnchanged {
		score = roundUp(math.Min(impact+exploitability, 10.0))
	} else {
		score = roundUp(math.Min(1.08*(impact+exploitability), 10.0))
	}

	return Result{
		Score:    score,
		Severity: classifySeverity(score),
		Vector:   formatVector(m),
		Breakdown: formatBreakdown(m, iss, impact, exploitability, score),
	}
}

// ParseVector parses a CVSS v3.1 vector string into BaseMetrics.
// Example: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
func ParseVector(vector string) (BaseMetrics, error) {
	m := BaseMetrics{}

	// Strip prefix
	vector = strings.TrimPrefix(vector, "CVSS:3.1/")
	vector = strings.TrimPrefix(vector, "CVSS:3.0/")

	parts := strings.Split(vector, "/")
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]

		switch key {
		case "AV":
			m.AttackVector = AttackVector(val)
		case "AC":
			m.AttackComplexity = AttackComplexity(val)
		case "PR":
			m.PrivilegesRequired = PrivilegesRequired(val)
		case "UI":
			m.UserInteraction = UserInteraction(val)
		case "S":
			m.Scope = Scope(val)
		case "C":
			m.ConfidentialityImpact = Impact(val)
		case "I":
			m.IntegrityImpact = Impact(val)
		case "A":
			m.AvailabilityImpact = Impact(val)
		}
	}

	// Validate required fields
	if m.AttackVector == "" || m.AttackComplexity == "" || m.PrivilegesRequired == "" ||
		m.UserInteraction == "" || m.Scope == "" || m.ConfidentialityImpact == "" ||
		m.IntegrityImpact == "" || m.AvailabilityImpact == "" {
		return m, fmt.Errorf("incomplete CVSS vector: missing required metrics")
	}

	return m, nil
}

// ── Metric Value Lookups (per CVSS v3.1 spec) ────────────

func attackVectorValue(av AttackVector) float64 {
	switch av {
	case AVNetwork:  return 0.85
	case AVAdjacent: return 0.62
	case AVLocal:    return 0.55
	case AVPhysical: return 0.20
	default:         return 0.85
	}
}

func attackComplexityValue(ac AttackComplexity) float64 {
	switch ac {
	case ACLow:  return 0.77
	case ACHigh: return 0.44
	default:     return 0.77
	}
}

func privReqValue(pr PrivilegesRequired, scope Scope) float64 {
	if scope == SChanged {
		switch pr {
		case PRNone: return 0.85
		case PRLow:  return 0.68
		case PRHigh: return 0.50
		default:     return 0.85
		}
	}
	switch pr {
	case PRNone: return 0.85
	case PRLow:  return 0.62
	case PRHigh: return 0.27
	default:     return 0.85
	}
}

func userInteractionValue(ui UserInteraction) float64 {
	switch ui {
	case UINone:     return 0.85
	case UIRequired: return 0.62
	default:         return 0.85
	}
}

func impactValue(i Impact) float64 {
	switch i {
	case ImpactHigh: return 0.56
	case ImpactLow:  return 0.22
	case ImpactNone: return 0.0
	default:         return 0.0
	}
}

// roundUp rounds up to one decimal place per the CVSS v3.1 specification.
// Uses "round up" semantics: roundUp(4.02) = 4.1, roundUp(4.00) = 4.0
func roundUp(val float64) float64 {
	intVal := math.Round(val * 100000)
	if int(intVal)%10000 == 0 {
		return intVal / 100000.0
	}
	return (math.Floor(intVal/10000) + 1) / 10.0
}

func classifySeverity(score float64) Severity {
	switch {
	case score == 0.0:
		return SeverityNone
	case score <= 3.9:
		return SeverityLow
	case score <= 6.9:
		return SeverityMedium
	case score <= 8.9:
		return SeverityHigh
	default:
		return SeverityCritical
	}
}

func formatVector(m BaseMetrics) string {
	return fmt.Sprintf("CVSS:3.1/AV:%s/AC:%s/PR:%s/UI:%s/S:%s/C:%s/I:%s/A:%s",
		m.AttackVector, m.AttackComplexity, m.PrivilegesRequired,
		m.UserInteraction, m.Scope,
		m.ConfidentialityImpact, m.IntegrityImpact, m.AvailabilityImpact,
	)
}

func formatBreakdown(m BaseMetrics, iss, impact, exploit, score float64) string {
	return fmt.Sprintf(
		"ISS=%.4f | Impact=%.4f | Exploitability=%.4f | Base=%.1f (%s)",
		iss, impact, exploit, score, classifySeverity(score),
	)
}

// ── Quick Constructors for Common Vulnerability Patterns ──

// CriticalRCE returns metrics for a typical unauthenticated remote code execution.
func CriticalRCE() BaseMetrics {
	return BaseMetrics{
		AttackVector:          AVNetwork,
		AttackComplexity:      ACLow,
		PrivilegesRequired:    PRNone,
		UserInteraction:       UINone,
		Scope:                 SUnchanged,
		ConfidentialityImpact: ImpactHigh,
		IntegrityImpact:       ImpactHigh,
		AvailabilityImpact:    ImpactHigh,
	}
}

// SQLInjection returns metrics for a typical SQL injection vulnerability.
func SQLInjection() BaseMetrics {
	return BaseMetrics{
		AttackVector:          AVNetwork,
		AttackComplexity:      ACLow,
		PrivilegesRequired:    PRNone,
		UserInteraction:       UINone,
		Scope:                 SUnchanged,
		ConfidentialityImpact: ImpactHigh,
		IntegrityImpact:       ImpactHigh,
		AvailabilityImpact:    ImpactNone,
	}
}

// XSS returns metrics for a typical reflected/stored XSS vulnerability.
func XSS() BaseMetrics {
	return BaseMetrics{
		AttackVector:          AVNetwork,
		AttackComplexity:      ACLow,
		PrivilegesRequired:    PRNone,
		UserInteraction:       UIRequired,
		Scope:                 SChanged,
		ConfidentialityImpact: ImpactLow,
		IntegrityImpact:       ImpactLow,
		AvailabilityImpact:    ImpactNone,
	}
}

// PrivilegeEscalation returns metrics for a typical local privilege escalation.
func PrivilegeEscalation() BaseMetrics {
	return BaseMetrics{
		AttackVector:          AVLocal,
		AttackComplexity:      ACLow,
		PrivilegesRequired:    PRLow,
		UserInteraction:       UINone,
		Scope:                 SUnchanged,
		ConfidentialityImpact: ImpactHigh,
		IntegrityImpact:       ImpactHigh,
		AvailabilityImpact:    ImpactHigh,
	}
}
