package opsec

import "sync"

// Manager tracks OPSEC/stealth state for the current session.
type Manager struct {
	mu      sync.RWMutex
	stealth bool
}

func NewManager() *Manager { return &Manager{} }

func (m *Manager) EnableStealth() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stealth = true
}

func (m *Manager) DisableStealth() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stealth = false
}

func (m *Manager) Toggle() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stealth = !m.stealth
	return m.stealth
}

func (m *Manager) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stealth
}

// StealthDirectives returns the OPSEC system prompt injection when stealth is active.
func (m *Manager) StealthDirectives() string {
	if !m.IsActive() {
		return ""
	}
	return `
--- OPSEC DIRECTIVES (STEALTH MODE ACTIVE) ---
- Introduce artificial jitter between commands (2-5 second random delays)
- Prefer passive recon methods over active scanning
- Avoid high-noise tools: masscan with fast rates, nuclei with many templates
- Use --rate-limit flags on all scanners
- Proxy all traffic where possible
- Do not generate more network traffic than strictly necessary
- Suppress any tool flags that generate excessive logs on the target
`
}
