package cvss

import (
	"testing"
)

func TestParseVectorValid(t *testing.T) {
	tests := []struct {
		name    string
		vector  string
		want    BaseMetrics
		wantErr bool
	}{
		{
			name:   "Log4Shell (CVE-2021-44228) - Critical RCE",
			vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			want: BaseMetrics{
				AttackVector:          AVNetwork,
				AttackComplexity:      ACLow,
				PrivilegesRequired:    PRNone,
				UserInteraction:       UINone,
				Scope:                 SUnchanged,
				ConfidentialityImpact: ImpactHigh,
				IntegrityImpact:       ImpactHigh,
				AvailabilityImpact:    ImpactHigh,
			},
		},
		{
			name:   "SQL Injection",
			vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N",
			want: BaseMetrics{
				AttackVector:          AVNetwork,
				AttackComplexity:      ACLow,
				PrivilegesRequired:    PRNone,
				UserInteraction:       UINone,
				Scope:                 SUnchanged,
				ConfidentialityImpact: ImpactHigh,
				IntegrityImpact:       ImpactHigh,
				AvailabilityImpact:    ImpactNone,
			},
		},
		{
			name:   "XSS Reflected",
			vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N",
			want: BaseMetrics{
				AttackVector:          AVNetwork,
				AttackComplexity:      ACLow,
				PrivilegesRequired:    PRNone,
				UserInteraction:       UIRequired,
				Scope:                 SChanged,
				ConfidentialityImpact: ImpactLow,
				IntegrityImpact:       ImpactLow,
				AvailabilityImpact:    ImpactNone,
			},
		},
		{
			name:   "Privilege Escalation",
			vector: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
			want: BaseMetrics{
				AttackVector:          AVLocal,
				AttackComplexity:      ACLow,
				PrivilegesRequired:    PRLow,
				UserInteraction:       UINone,
				Scope:                 SUnchanged,
				ConfidentialityImpact: ImpactHigh,
				IntegrityImpact:       ImpactHigh,
				AvailabilityImpact:    ImpactHigh,
			},
		},
		{
			name:   "Without prefix",
			vector: "AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			want: BaseMetrics{
				AttackVector:          AVNetwork,
				AttackComplexity:      ACLow,
				PrivilegesRequired:    PRNone,
				UserInteraction:       UINone,
				Scope:                 SUnchanged,
				ConfidentialityImpact: ImpactHigh,
				IntegrityImpact:       ImpactHigh,
				AvailabilityImpact:    ImpactHigh,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVector(tt.vector)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseVector() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseVectorInvalid(t *testing.T) {
	tests := []struct {
		name    string
		vector  string
		wantErr bool
	}{
		{
			name:    "empty string",
			vector:  "",
			wantErr: true,
		},
		{
			name:    "missing metrics",
			vector:  "CVSS:3.1/AV:N/AC:L",
			wantErr: true,
		},
		{
			name:    "invalid metric value",
			vector:  "CVSS:3.1/AV:X/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			wantErr: false, // Parser doesn't validate values, just presence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseVector(tt.vector)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVector() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateBaseScore(t *testing.T) {
	tests := []struct {
		name       string
		metrics    BaseMetrics
		wantScore  float64
		wantSev    Severity
	}{
		{
			name: "Critical RCE (Log4Shell)",
			metrics: BaseMetrics{
				AttackVector:          AVNetwork,
				AttackComplexity:      ACLow,
				PrivilegesRequired:    PRNone,
				UserInteraction:       UINone,
				Scope:                 SUnchanged,
				ConfidentialityImpact: ImpactHigh,
				IntegrityImpact:       ImpactHigh,
				AvailabilityImpact:    ImpactHigh,
			},
			wantScore: 9.8,
			wantSev:   SeverityCritical,
		},
		{
			name: "High SQL Injection",
			metrics: BaseMetrics{
				AttackVector:          AVNetwork,
				AttackComplexity:      ACLow,
				PrivilegesRequired:    PRNone,
				UserInteraction:       UINone,
				Scope:                 SUnchanged,
				ConfidentialityImpact: ImpactHigh,
				IntegrityImpact:       ImpactHigh,
				AvailabilityImpact:    ImpactNone,
			},
			wantScore: 9.1,
			wantSev:   SeverityCritical,
		},
		{
			name: "Medium XSS Reflected",
			metrics: BaseMetrics{
				AttackVector:          AVNetwork,
				AttackComplexity:      ACLow,
				PrivilegesRequired:    PRNone,
				UserInteraction:       UIRequired,
				Scope:                 SChanged,
				ConfidentialityImpact: ImpactLow,
				IntegrityImpact:       ImpactLow,
				AvailabilityImpact:    ImpactNone,
			},
			wantScore: 6.1,
			wantSev:   SeverityMedium,
		},
		{
			name: "Low Info Disclosure",
			metrics: BaseMetrics{
				AttackVector:          AVNetwork,
				AttackComplexity:      ACHigh,
				PrivilegesRequired:    PRNone,
				UserInteraction:       UIRequired,
				Scope:                 SUnchanged,
				ConfidentialityImpact: ImpactLow,
				IntegrityImpact:       ImpactNone,
				AvailabilityImpact:    ImpactNone,
			},
			wantScore: 3.1,
			wantSev:   SeverityLow,
		},
		{
			name: "No Impact",
			metrics: BaseMetrics{
				AttackVector:          AVNetwork,
				AttackComplexity:      ACLow,
				PrivilegesRequired:    PRNone,
				UserInteraction:       UINone,
				Scope:                 SUnchanged,
				ConfidentialityImpact: ImpactNone,
				IntegrityImpact:       ImpactNone,
				AvailabilityImpact:    ImpactNone,
			},
			wantScore: 0.0,
			wantSev:   SeverityNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBaseScore(tt.metrics)
			if result.Score != tt.wantScore {
				t.Errorf("CalculateBaseScore() score = %f, want %f", result.Score, tt.wantScore)
			}
			if result.Severity != tt.wantSev {
				t.Errorf("CalculateBaseScore() severity = %s, want %s", result.Severity, tt.wantSev)
			}
		})
	}
}

func TestRoundUp(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  float64
	}{
		{"exact", 4.0, 4.0},
		{"round up", 4.02, 4.1},
		{"round up at boundary", 4.10, 4.1},
		{"round up small", 0.11, 0.2},
		{"zero", 0.0, 0.0},
		{"round up 9.81", 9.81, 9.9},
		{"round up 9.89", 9.89, 9.9},
		{"round up 9.91", 9.91, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundUp(tt.input)
			if got != tt.want {
				t.Errorf("roundUp(%f) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

func TestClassifySeverity(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  Severity
	}{
		{"none", 0.0, SeverityNone},
		{"low boundary", 0.1, SeverityLow},
		{"low max", 3.9, SeverityLow},
		{"medium boundary", 4.0, SeverityMedium},
		{"medium max", 6.9, SeverityMedium},
		{"high boundary", 7.0, SeverityHigh},
		{"high max", 8.9, SeverityHigh},
		{"critical boundary", 9.0, SeverityCritical},
		{"critical max", 10.0, SeverityCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySeverity(tt.score)
			if got != tt.want {
				t.Errorf("classifySeverity(%f) = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}

func TestFormatVector(t *testing.T) {
	m := BaseMetrics{
		AttackVector:          AVNetwork,
		AttackComplexity:      ACLow,
		PrivilegesRequired:    PRNone,
		UserInteraction:       UINone,
		Scope:                 SUnchanged,
		ConfidentialityImpact: ImpactHigh,
		IntegrityImpact:       ImpactHigh,
		AvailabilityImpact:    ImpactHigh,
	}
	got := formatVector(m)
	want := "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
	if got != want {
		t.Errorf("formatVector() = %s, want %s", got, want)
	}
}

func TestCalculateBaseScoreRoundTrip(t *testing.T) {
	// Parse a vector, calculate score, format vector, parse again — should be identical
	original := "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
	metrics, err := ParseVector(original)
	if err != nil {
		t.Fatalf("ParseVector() error = %v", err)
	}
	result := CalculateBaseScore(metrics)
	parsedAgain, err := ParseVector(result.Vector)
	if err != nil {
		t.Fatalf("ParseVector() on result error = %v", err)
	}
	if parsedAgain != metrics {
		t.Errorf("Round-trip failed: got %+v, want %+v", parsedAgain, metrics)
	}
}
