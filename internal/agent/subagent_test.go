package agent

import (
	"testing"
	"time"
)

func TestSubagentManagerExecuteParallel(t *testing.T) {
	// Test with a simple mock - just verify the structure works
	sm := NewSubagentManager(nil, nil, 2)
	if sm.maxConc != 2 {
		t.Errorf("maxConc = %d, want 2", sm.maxConc)
	}
}

func TestStandardReconTasks(t *testing.T) {
	tasks := StandardReconTasks("10.0.0.1")
	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	ids := make(map[string]bool)
	for _, task := range tasks {
		ids[task.ID] = true
		if task.Args["target"] != "10.0.0.1" {
			t.Errorf("Task %s has wrong target: %v", task.ID, task.Args["target"])
		}
	}

	if !ids["port_scan"] || !ids["subdomain_enum"] || !ids["web_recon"] {
		t.Errorf("Missing expected tasks: %v", ids)
	}
}

func TestFullWebReconTasks(t *testing.T) {
	tasks := FullWebReconTasks("example.com")
	if len(tasks) != 5 {
		t.Fatalf("Expected 5 tasks, got %d", len(tasks))
	}

	// Check dependencies exist
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			found := false
			for _, other := range tasks {
				if other.ID == dep {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Task %s depends on non-existent task %s", task.ID, dep)
			}
		}
	}
}

func TestCloudTasks(t *testing.T) {
	tasks := CloudTasks("aws-target.com")
	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}
}

func TestFormatResultsForLLM(t *testing.T) {
	results := []SubagentResult{
		{TaskID: "scan1", Tool: "nmap", Output: "80/open", Duration: 5 * time.Second},
		{TaskID: "scan2", Tool: "httpx", Output: "Status: 200", Duration: 3 * time.Second},
	}

	output := FormatResultsForLLM(results)
	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestFormatResultsForLMEmpty(t *testing.T) {
	output := FormatResultsForLLM(nil)
	if output != "" {
		t.Errorf("Expected empty for nil results, got %q", output)
	}
}

func TestExtractPortFindings(t *testing.T) {
	output := "80/tcp open  http\n443/tcp open  https\n22/tcp closed ssh"
	findings := extractPortFindings(output)

	if len(findings) < 2 {
		t.Errorf("Expected at least 2 port findings, got %d", len(findings))
	}
}

func TestExtractServiceFindings(t *testing.T) {
	output := "Service: http\nhttp/nginx 1.18\nService: ssh"
	findings := extractServiceFindings(output)

	if len(findings) == 0 {
		t.Error("Expected service findings")
	}
}
