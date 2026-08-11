// Package planner provides a resumable task tree for DrogonClaw engagements.
//
// It deliberately reuses the existing memory.Graph as its storage substrate:
// planning tasks are Graph nodes (label LabelTask) and dependencies are PARENT
// edges. This means (a) no parallel tree structure is introduced and (b) resume
// is free — Graph already persists to data/graph_<session>.json and reloads it
// on NewGraph, so a re-run with the same session ID resumes from completed tasks.
//
// The ordering in Next() is an EGATS-lite stand-in for PentestGPT V2's
// difficulty-aware planning: it prefers shallow, high-exploitability classes and
// only schedules a task once its parent is done.
package planner

import (
	"sort"

	"github.com/0xP4X/drogonclaw-go/internal/memory"
)

// LabelTask marks a planning task node in the intelligence graph.
const LabelTask memory.NodeLabel = "Task"

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusActive  TaskStatus = "active"
	StatusDone    TaskStatus = "done"
	StatusBlocked TaskStatus = "blocked"
)

// Task is a single planning node projected from the graph.
type Task struct {
	ID       string
	Title    string
	Class    string // owasp class, "sast", "recon", "verify", or "phase"
	Depth    int
	ParentID string
	Status   TaskStatus
	Evidence string
	Score    float64
}

// Planner drives a task tree stored in a memory.Graph.
type Planner struct {
	graph  *memory.Graph
	rootID string
}

func New(graph *memory.Graph) *Planner {
	return &Planner{graph: graph}
}

// HasPlan reports whether a task tree already exists for this session (used to
// skip rebuilding on resume).
func (p *Planner) HasPlan() bool {
	for _, n := range p.graph.GetNodes() {
		if n.Label == LabelTask {
			return true
		}
	}
	return false
}

// BuildWhiteboxPlan creates the standard five-phase white-box task tree. It is
// a no-op if HasPlan() is already true, preventing duplicate tasks on resume.
func (p *Planner) BuildWhiteboxPlan(targetURL, repoPath string) {
	if p.HasPlan() {
		return
	}
	root := p.add("White-box assessment", "phase", 0, "")
	p.rootID = root
	if repoPath != "" {
		p.add("Pre-Recon: source SAST", "sast", 1, root)
	}
	if targetURL != "" {
		p.add("Recon: attack surface", "recon", 1, root)
		p.add("Vuln: Injection", "injection", 1, root)
		p.add("Vuln: XSS", "xss", 1, root)
		p.add("Vuln: SSRF", "ssrf", 1, root)
		p.add("Vuln: AuthN", "authn", 1, root)
		p.add("Vuln: AuthZ", "authz", 1, root)
		p.add("Exploit: verify findings", "verify", 2, root)
	}
	// The phase root is immediately satisfied so its children become schedulable.
	p.MarkDone(root, "")
}

func (p *Planner) add(title, class string, depth int, parentID string) string {
	id := "task-" + itoa(p.count()+1)
	props := map[string]any{
		"title":  title,
		"class":  class,
		"depth":  depth,
		"status": string(StatusPending),
	}
	p.graph.AddNode(&memory.Node{ID: id, Label: LabelTask, Properties: props})
	if parentID != "" {
		p.graph.AddEdge(&memory.Edge{SourceID: parentID, TargetID: id, Relationship: "PARENT"})
	}
	return id
}

func (p *Planner) count() int {
	n := 0
	for _, node := range p.graph.GetNodes() {
		if node.Label == LabelTask {
			n++
		}
	}
	return n
}

// TaskByClass returns the first task with the given class, or nil.
func (p *Planner) TaskByClass(class string) *Task {
	for _, t := range p.tasks() {
		if t.Class == class {
			return &t
		}
	}
	return nil
}

// MarkDone records a task as completed with optional evidence, persisting via
// the graph's existing save path.
func (p *Planner) MarkDone(id, evidence string) {
	for _, node := range p.graph.GetNodes() {
		if node.ID != id {
			continue
		}
		node.Properties["status"] = string(StatusDone)
		if evidence != "" {
			node.Properties["evidence"] = evidence
		}
		p.graph.AddNode(node) // re-add to persist
		return
	}
}

// Pending returns the number of not-yet-done tasks.
func (p *Planner) Pending() int {
	n := 0
	for _, t := range p.tasks() {
		if t.Status != StatusDone {
			n++
		}
	}
	return n
}

// Next returns the highest-priority schedulable task (parent done, not itself
// done), using EGATS-lite scoring. Returns nil when nothing is schedulable.
func (p *Planner) Next() *Task {
	tasks := p.tasks()
	var best *Task
	bestScore := -1.0
	for i := range tasks {
		t := &tasks[i]
		if t.Status == StatusDone || t.Status == StatusBlocked {
			continue
		}
		if t.ParentID != "" {
			if parent := byID(tasks, t.ParentID); parent != nil && parent.Status != StatusDone {
				continue // dependency not satisfied
			}
		}
		score := p.score(t)
		if score > bestScore {
			bestScore = score
			best = t
		}
	}
	return best
}

// score is the EGATS-lite heuristic: higher exploitability class and shallower
// depth rank first; blocked/active states are excluded by Next().
func (p *Planner) score(t *Task) float64 {
	base := classPriority[t.Class]
	base -= 0.1 * float64(t.Depth)
	return base
}

var classPriority = map[string]float64{
	"injection": 1.0,
	"authn":     0.95,
	"xss":       0.9,
	"ssrf":      0.85,
	"authz":     0.8,
	"sast":      0.6,
	"recon":     0.5,
	"verify":    0.4,
	"phase":     0.0,
}

func (p *Planner) tasks() []Task {
	var out []Task
	for _, node := range p.graph.GetNodes() {
		if node.Label != LabelTask {
			continue
		}
		out = append(out, Task{
			ID:       node.ID,
			Title:    propStr(node, "title"),
			Class:    propStr(node, "class"),
			Depth:    propInt(node, "depth"),
			ParentID: parentOf(p.graph, node.ID),
			Status:   TaskStatus(propStr(node, "status")),
			Evidence: propStr(node, "evidence"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func parentOf(g *memory.Graph, id string) string {
	for _, e := range g.Edges() {
		if e.TargetID == id && e.Relationship == "PARENT" {
			return e.SourceID
		}
	}
	return ""
}

func byID(tasks []Task, id string) *Task {
	for i := range tasks {
		if tasks[i].ID == id {
			return &tasks[i]
		}
	}
	return nil
}

func propStr(n *memory.Node, key string) string {
	if v, ok := n.Properties[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func propInt(n *memory.Node, key string) int {
	if v, ok := n.Properties[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return 0
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
