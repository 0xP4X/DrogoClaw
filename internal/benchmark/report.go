package benchmark

import (
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
	md.WriteString("| ID | Class | Solved | Flag | Duration |\n")
	md.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, o := range s.Outcomes {
		flag := o.Flag
		if len(flag) > 40 {
			flag = flag[:37] + "..."
		}
		fmt.Fprintf(md, "| %s | %s | %t | %s | %s |\n", o.ID, o.Class, o.Solved, flag, o.Duration)
	}
	md.WriteString("\n")

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
	return mdPath, nil
}

