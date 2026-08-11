package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type NodeLabel string

const (
	LabelTarget        NodeLabel = "Target"
	LabelAsset         NodeLabel = "Asset"
	LabelPort          NodeLabel = "Port"
	LabelService       NodeLabel = "Service"
	LabelVulnerability NodeLabel = "Vulnerability"
	LabelCredential    NodeLabel = "Credential"
	LabelFlag          NodeLabel = "Flag"
	LabelTask          NodeLabel = "Task"
)

type Node struct {
	ID         string         `json:"id"`
	Label      NodeLabel      `json:"label"`
	Properties map[string]any `json:"properties"`
	Timestamp  int64          `json:"timestamp"`
}

type Edge struct {
	SourceID     string `json:"sourceId"`
	TargetID     string `json:"targetId"`
	Relationship string `json:"relationship"`
}

type OperatorProfile struct {
	Name       string `json:"name"`
	SkillLevel string `json:"skillLevel,omitempty"`
	Prefs      string `json:"preferences,omitempty"`
}

type AgentProfile struct {
	Name string `json:"name"`
}

type graphData struct {
	Nodes           []*Node          `json:"nodes"`
	Edges           []*Edge          `json:"edges"`
	OperatorProfile *OperatorProfile `json:"operatorProfile,omitempty"`
	AgentProfile    *AgentProfile    `json:"agentProfile,omitempty"`
}

// Graph is the in-memory intelligence map for a DrogonClaw session.
type Graph struct {
	mu              sync.RWMutex
	nodes           map[string]*Node
	edges           []*Edge
	operatorProfile *OperatorProfile
	agentProfile    *AgentProfile
	dbPath          string
}

func NewGraph(sessionID string) *Graph {
	dataDir := filepath.Join("data")
	_ = os.MkdirAll(dataDir, 0755)

	g := &Graph{
		nodes:  make(map[string]*Node),
		dbPath: filepath.Join(dataDir, fmt.Sprintf("graph_%s.json", sessionID)),
	}
	g.load()
	return g
}

func (g *Graph) AddNode(node *Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	node.Timestamp = time.Now().UnixMilli()
	g.nodes[node.ID] = node
	g.save()
}

func (g *Graph) AddEdge(edge *Edge) {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Deduplicate
	for _, e := range g.edges {
		if e.SourceID == edge.SourceID && e.TargetID == edge.TargetID && e.Relationship == edge.Relationship {
			return
		}
	}
	g.edges = append(g.edges, edge)
	g.save()
}

func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

func (g *Graph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.edges)
}

func (g *Graph) GetNodes() map[string]*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	nodes := make(map[string]*Node, len(g.nodes))
	for id, node := range g.nodes {
		nodes[id] = node
	}
	return nodes
}

// Edges returns a copy of the current relationship set.
func (g *Graph) Edges() []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*Edge, len(g.edges))
	copy(out, g.edges)
	return out
}

func (g *Graph) LabelCounts() map[NodeLabel]int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	counts := make(map[NodeLabel]int)
	for _, node := range g.nodes {
		counts[node.Label]++
	}
	return counts
}

func (g *Graph) RelationshipCounts() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	counts := make(map[string]int)
	for _, edge := range g.edges {
		counts[edge.Relationship]++
	}
	return counts
}

func (g *Graph) GetOperatorProfile() *OperatorProfile {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.operatorProfile
}

func (g *Graph) GetAgentProfile() *AgentProfile {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.agentProfile
}

func (g *Graph) UpdateOperatorProfile(p *OperatorProfile) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.operatorProfile == nil {
		g.operatorProfile = &OperatorProfile{}
	}
	if p.Name != "" {
		g.operatorProfile.Name = p.Name
	}
	if p.SkillLevel != "" {
		g.operatorProfile.SkillLevel = p.SkillLevel
	}
	if p.Prefs != "" {
		g.operatorProfile.Prefs = p.Prefs
	}
	g.save()
}

func (g *Graph) UpdateAgentProfile(p *AgentProfile) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.agentProfile == nil {
		g.agentProfile = &AgentProfile{}
	}
	if p.Name != "" {
		g.agentProfile.Name = p.Name
	}
	g.save()
}

func (g *Graph) GetFullJSON() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	nodes := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	data := graphData{Nodes: nodes, Edges: g.edges, OperatorProfile: g.operatorProfile, AgentProfile: g.agentProfile}
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}

// Snapshot returns a compact, human-readable view of the graph for injection
// into the agent's context. Without this the graph is write-only: the agent
// stores entities but never sees them again, so memory never actually
// influences behaviour.
func (g *Graph) Snapshot() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("MEMORY GRAPH STATE\n")

	if g.operatorProfile != nil && g.operatorProfile.Name != "" {
		sb.WriteString("Operator: " + g.operatorProfile.Name + "\n")
	} else {
		sb.WriteString("Operator: (unknown)\n")
	}

	if len(g.nodes) == 0 {
		sb.WriteString("Entities: none recorded yet.\n")
		return strings.TrimRight(sb.String(), "\n")
	}

	labelCounts := make(map[NodeLabel]int)
	for _, n := range g.nodes {
		labelCounts[n.Label]++
	}
	labels := make([]string, 0, len(labelCounts))
	for l := range labelCounts {
		labels = append(labels, string(l))
	}
	sort.Strings(labels)
	var lcParts []string
	for _, l := range labels {
		lcParts = append(lcParts, fmt.Sprintf("%s=%d", l, labelCounts[NodeLabel(l)]))
	}
	sb.WriteString(fmt.Sprintf("Entities: %d (%s)\n", len(g.nodes), strings.Join(lcParts, ", ")))
	sb.WriteString(fmt.Sprintf("Relationships: %d\n", len(g.edges)))

	// Show the most recent entities so the agent can reason about what it knows.
	recent := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		recent = append(recent, n)
	}
	sort.Slice(recent, func(i, j int) bool { return recent[i].Timestamp > recent[j].Timestamp })
	limit := len(recent)
	if limit > 12 {
		limit = 12
	}
	for _, n := range recent[:limit] {
		sb.WriteString(fmt.Sprintf("- %s [%s]", n.ID, n.Label))
		if len(n.Properties) > 0 {
			var props []string
			for k, v := range n.Properties {
				props = append(props, fmt.Sprintf("%s=%v", k, v))
			}
			sb.WriteString(" (" + strings.Join(props, ", ") + ")")
		}
		sb.WriteString("\n")
	}

	if len(g.edges) > 0 {
		limit := len(g.edges)
		if limit > 12 {
			limit = 12
		}
		for _, e := range g.edges[:limit] {
			sb.WriteString(fmt.Sprintf("- %s --%s--> %s\n", e.SourceID, e.Relationship, e.TargetID))
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

func (g *Graph) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes = make(map[string]*Node)
	g.edges = nil
	g.save()
}

func (g *Graph) save() {
	nodes := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	data := graphData{Nodes: nodes, Edges: g.edges, OperatorProfile: g.operatorProfile, AgentProfile: g.agentProfile}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	// Write then rename so a crash can leave at most the previous complete
	// graph, never a partially written JSON document.
	tmp := g.dbPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err == nil {
		_ = os.Rename(tmp, g.dbPath)
	}
}

func (g *Graph) load() {
	b, err := os.ReadFile(g.dbPath)
	if err != nil {
		return // first session — empty graph
	}
	var data graphData
	if err := json.Unmarshal(b, &data); err != nil {
		return
	}
	for _, n := range data.Nodes {
		g.nodes[n.ID] = n
	}
	g.edges = data.Edges
	g.operatorProfile = data.OperatorProfile
	g.agentProfile = data.AgentProfile
}
