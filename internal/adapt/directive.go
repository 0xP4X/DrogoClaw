package adapt

import (
	"fmt"
	"strings"
	"sync"
)

// DirectiveStore holds runtime operator directives that modify the AI's behavior.
type DirectiveStore struct {
	mu         sync.RWMutex
	directives map[string]string
}

var GlobalDirectives = &DirectiveStore{
	directives: make(map[string]string),
}

// Set adds or updates a directive.
func (d *DirectiveStore) Set(key, value string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.directives[key] = value
}

// Get retrieves a directive value.
func (d *DirectiveStore) Get(key string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	v, ok := d.directives[key]
	return v, ok
}

// Delete removes a directive.
func (d *DirectiveStore) Delete(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.directives, key)
}

// BuildDirectiveBlock returns all active directives as a formatted system prompt block.
func (d *DirectiveStore) BuildDirectiveBlock() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.directives) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n--- RUNTIME SELF-DIRECTIVES (set by AI during operation) ---\n")
	for k, v := range d.directives {
		sb.WriteString(fmt.Sprintf("• %s: %s\n", k, v))
	}
	return sb.String()
}

// All returns a copy of all directives.
func (d *DirectiveStore) All() map[string]string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make(map[string]string, len(d.directives))
	for k, v := range d.directives {
		out[k] = v
	}
	return out
}
