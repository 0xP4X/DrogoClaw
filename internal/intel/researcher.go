package intel

import (
	"fmt"
	"strings"
)

// ResearchResult summarizes a multi-step deep research operation.
type ResearchResult struct {
	Topic    string
	Summary  string
	Sources  []string
	RawPages []string
}

// DeepResearch searches for a topic, fetches the top N results, and synthesizes
// a combined text summary that can be fed directly to the LLM for reasoning.
func DeepResearch(topic, braveAPIKey string, depth int) (*ResearchResult, error) {
	if depth <= 0 {
		depth = 5
	}

	results, err := Search(topic, braveAPIKey, depth)
	if err != nil {
		return nil, fmt.Errorf("search phase failed: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no search results found for: %s", topic)
	}

	var pages []string
	var sources []string
	var failedURLs []string

	for _, r := range results {
		sources = append(sources, fmt.Sprintf("%s — %s", r.Title, r.URL))
		content, err := FetchURL(r.URL)
		if err != nil {
			failedURLs = append(failedURLs, r.URL)
			// Fall back to the snippet if the full fetch fails
			pages = append(pages, fmt.Sprintf("### %s\n%s", r.Title, r.Snippet))
			continue
		}
		pages = append(pages, fmt.Sprintf("### %s\nSource: %s\n\n%s", r.Title, r.URL, content))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Deep Research: %s\n\n", topic))
	sb.WriteString(fmt.Sprintf("Sources consulted: %d | Failed fetches: %d\n\n", len(results), len(failedURLs)))
	sb.WriteString("---\n\n")
	for _, page := range pages {
		sb.WriteString(page)
		sb.WriteString("\n\n---\n\n")
	}

	return &ResearchResult{
		Topic:    topic,
		Summary:  sb.String(),
		Sources:  sources,
		RawPages: pages,
	}, nil
}
