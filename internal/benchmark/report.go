package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Write saves the summary as a Markdown report and a JSON file inside outDir.
// It returns the path to the Markdown report.
func (s *Summary) Write(outDir string) (string, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	md := new(strings.Builder)
	md.WriteString("# DrogonClaw Benchmark Report\n\n")
	fmt.Fprintf(md, "- **Set:** %s\n", s.Set)
	fmt.Fprintf(md, "- **Run:** %s\n", s.RunAt.Format(time.RFC3339))
	fmt.Fprintf(md, "- **Solved:** %d / %d\n", s.Solved, s.Total)
	fmt.Fprintf(md, "- **Success rate:** %.1f%%\n", s.SuccessRate)
	fmt.Fprintf(md, "- **Avg duration:** %s\n\n", s.AvgDuration)

	if len(s.ByClass) > 0 {
		md.WriteString("## By class\n\n")
		md.WriteString("| Class | Solved | Rate |\n")
		md.WriteString("| --- | --- | --- |\n")
		for class, stat := range s.ByClass {
			fmt.Fprintf(md, "| %s | %d/%d | %.0f%% |\n", class, stat.Solved, stat.Total, stat.Rate)
		}
		md.WriteString("\n")
	}

	md.WriteString("## Challenges\n\n")
	md.WriteString("| ID | Class | Solved | Flag | Duration | Cost |\n")
	md.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, o := range s.Outcomes {
		flag := o.Flag
		if len(flag) > 40 {
			flag = flag[:37] + "..."
		}
		cost := fmt.Sprintf("$%.4f", o.CostUSD)
		fmt.Fprintf(md, "| %s | %s | %t | %s | %s | %s |\n", o.ID, o.Class, o.Solved, flag, o.Duration, cost)
	}
	md.WriteString("\n")

	// Add detailed execution logs for each challenge
	md.WriteString("## Detailed Execution Logs\n\n")
	for _, o := range s.Outcomes {
		if o.Solved || o.Err != "" {
			md.WriteString(fmt.Sprintf("### %s\n\n", o.ID))
			md.WriteString(fmt.Sprintf("- **Status:** %t\n", o.Solved))
			md.WriteString(fmt.Sprintf("- **Duration:** %s\n", o.Duration))
			md.WriteString(fmt.Sprintf("- **Cost:** $%.4f\n", o.CostUSD))
			if o.Flag != "" {
				md.WriteString(fmt.Sprintf("- **Flag:** `%s`\n", o.Flag))
			}
			if o.Err != "" {
				md.WriteString(fmt.Sprintf("- **Error:** %s\n", o.Err))
			}
			md.WriteString("\n")
		}
	}

	// Add historical comparison if available
	historyPath := filepath.Join(outDir, "history.json")
	if historicalData, err := loadHistoricalData(historyPath); err == nil && len(historicalData) > 0 {
		md.WriteString("## Historical Comparison\n\n")
		md.WriteString("| Run | Solved | Rate | Avg Duration |\n")
		md.WriteString("| --- | --- | --- | --- |\n")
		
		// Add current run
		md.WriteString(fmt.Sprintf("| %s (current) | %d/%d | %.1f%% | %s |\n", 
			s.RunAt.Format("2006-01-02 15:04"), s.Solved, s.Total, s.SuccessRate, s.AvgDuration))
		
		// Add previous runs (last 5)
		start := max(0, len(historicalData)-5)
		for _, hist := range historicalData[start:] {
			md.WriteString(fmt.Sprintf("| %s | %d/%d | %.1f%% | %s |\n",
				hist.RunAt.Format("2006-01-02 15:04"), hist.Solved, hist.Total, hist.SuccessRate, hist.AvgDuration))
		}
		md.WriteString("\n")
	}

	// Add Mermaid charts
	md.WriteString("## Execution Flow\n\n")
	md.WriteString("```mermaid\n")
	md.WriteString("graph TD\n")
	for i, o := range s.Outcomes {
		status := "FAIL"
		if o.Solved {
			status = "PASS"
		}
		md.WriteString(fmt.Sprintf("    A%d[%s] --> B%d[%s]\n", i, o.ID, i, status))
	}
	md.WriteString("```\n\n")

	// Add success rate pie chart
	md.WriteString("```mermaid\n")
	md.WriteString("pie title Success Rate by Class\n")
	for class, stat := range s.ByClass {
		md.WriteString(fmt.Sprintf("    \"%s\" : %d\n", class, stat.Solved))
	}
	if s.Total > s.Solved {
		md.WriteString(fmt.Sprintf("    \"Failed\" : %d\n", s.Total-s.Solved))
	}
	md.WriteString("```\n\n")

	mdPath := filepath.Join(outDir, "report.md")
	if err := os.WriteFile(mdPath, []byte(md.String()), 0644); err != nil {
		return "", err
	}

	jsonPath := filepath.Join(outDir, "results.json")
	data, err := jsonMarshalIndent(s)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return "", err
	}

	// Save to historical data
	saveToHistory(historyPath, s)

	return mdPath, nil
}

// HistoricalRun represents a historical benchmark run
type HistoricalRun struct {
	RunAt       time.Time `json:"runAt"`
	Set         string    `json:"set"`
	Total       int       `json:"total"`
	Solved      int       `json:"solved"`
	SuccessRate float64   `json:"successRate"`
	AvgDuration string    `json:"avgDuration"`
}

// loadHistoricalData loads historical benchmark runs
func loadHistoricalData(path string) ([]HistoricalRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var runs []HistoricalRun
	if err := json.Unmarshal(data, &runs); err != nil {
		return nil, err
	}

	return runs, nil
}

// saveToHistory saves the current run to historical data
func saveToHistory(path string, s *Summary) {
	runs, _ := loadHistoricalData(path)
	
	// Add current run
	runs = append(runs, HistoricalRun{
		RunAt:       s.RunAt,
		Set:         s.Set,
		Total:       s.Total,
		Solved:      s.Solved,
		SuccessRate: s.SuccessRate,
		AvgDuration: s.AvgDuration,
	})
	
	// Keep only last 20 runs
	if len(runs) > 20 {
		runs = runs[len(runs)-20:]
	}
	
	data, err := jsonMarshalIndent(runs)
	if err != nil {
		return
	}
	
	os.WriteFile(path, data, 0644)
}

// CompareRuns compares two benchmark summaries
func CompareRuns(current, previous *Summary) string {
	var sb strings.Builder
	
	sb.WriteString("# Benchmark Comparison\n\n")
	sb.WriteString(fmt.Sprintf("Current run: %s\n", current.RunAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Previous run: %s\n\n", previous.RunAt.Format(time.RFC3339)))
	
	// Success rate change
	rateDiff := current.SuccessRate - previous.SuccessRate
	status := "improved"
	if rateDiff < 0 {
		status = "regressed"
	}
	sb.WriteString(fmt.Sprintf("**Success Rate:** %.1f%% → %.1f%% (%.1f%% %s)\n\n",
		previous.SuccessRate, current.SuccessRate, rateDiff, status))
	
	// Per-challenge comparison
	sb.WriteString("## Challenge Changes\n\n")
	sb.WriteString("| Challenge | Previous | Current | Change |\n")
	sb.WriteString("| --- | --- | --- | --- |\n")
	
	prevMap := make(map[string]Outcome)
	for _, o := range previous.Outcomes {
		prevMap[o.ID] = o
	}
	
	for _, curr := range current.Outcomes {
		prev, exists := prevMap[curr.ID]
		if exists {
			change := "✓ Same"
			if curr.Solved && !prev.Solved {
				change = "✓ Fixed"
			} else if !curr.Solved && prev.Solved {
				change = "✗ Regression"
			}
			sb.WriteString(fmt.Sprintf("| %s | %t | %t | %s |\n", curr.ID, prev.Solved, curr.Solved, change))
		} else {
			sb.WriteString(fmt.Sprintf("| %s | N/A | %t | New |\n", curr.ID, curr.Solved))
		}
	}
	sb.WriteString("\n")
	
	return sb.String()
}

