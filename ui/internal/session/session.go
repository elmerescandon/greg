package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Session struct {
	ID             string  `json:"id"`
	Dir            string  `json:"dir"`
	Started        string  `json:"started"`
	Ended          string  `json:"ended,omitempty"`
	Status         string  `json:"status"`
	ClaudeSession  string  `json:"claude_session_id,omitempty"`
	Model          string  `json:"model,omitempty"`
	OutputTokens   int     `json:"output_tokens,omitempty"`
	CostUSD        float64 `json:"cost_usd,omitempty"`
}

func GregHome() string {
	return filepath.Join(os.Getenv("HOME"), ".greg")
}

func SessionsFile() string {
	return filepath.Join(GregHome(), "sessions.json")
}

func HistoryFile() string {
	return filepath.Join(GregHome(), "history.json")
}

func LoadFinishedSessions() ([]Session, error) {
	data, err := os.ReadFile(HistoryFile())
	if err != nil {
		return nil, err
	}
	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func MailboxDir() string {
	return filepath.Join(GregHome(), "mailbox")
}

func LoadSessions() ([]Session, error) {
	data, err := os.ReadFile(SessionsFile())
	if err != nil {
		return nil, err
	}
	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func SaveSessions(sessions []Session) error {
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(SessionsFile(), data, 0644)
}

func Spawn(vault string) (Session, error) {
	b := make([]byte, 4)
	rand.Read(b)
	id := "greg-" + hex.EncodeToString(b)
	started := time.Now().Format("2006-01-02 15:04:05")

	mbDir := filepath.Join(MailboxDir(), id)
	os.MkdirAll(mbDir, 0755)
	os.WriteFile(filepath.Join(mbDir, "inbox.md"), []byte{}, 0644)

	s := Session{ID: id, Dir: vault, Started: started, Status: "active"}

	sessions, _ := LoadSessions()
	sessions = append(sessions, s)
	if err := SaveSessions(sessions); err != nil {
		return s, err
	}
	return s, nil
}

// ArchiveTaskSessions moves the given session IDs from sessions.json to history.json as finished.
func ArchiveTaskSessions(ids []string) {
	if len(ids) == 0 {
		return
	}
	toArchive := map[string]bool{}
	for _, id := range ids {
		toArchive[id] = true
	}
	sessions, err := LoadSessions()
	if err != nil {
		return
	}
	history, _ := LoadFinishedSessions()
	ended := time.Now().Format("2006-01-02 15:04:05")
	var remaining []Session
	for _, s := range sessions {
		if toArchive[s.ID] {
			s.Status = "finished"
			s.Ended = ended
			history = append(history, s)
		} else {
			remaining = append(remaining, s)
		}
	}
	if len(remaining) == len(sessions) {
		return
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(HistoryFile(), data, 0644); err != nil {
		return
	}
	SaveSessions(remaining)
}

func UpdateModel(gregID, model string) {
	sessions, err := LoadSessions()
	if err != nil {
		return
	}
	for i := range sessions {
		if sessions[i].ID == gregID {
			sessions[i].Model = model
			break
		}
	}
	SaveSessions(sessions)
}

func UpdateClaudeSession(gregID, claudeSessionID string) {
	sessions, err := LoadSessions()
	if err != nil {
		return
	}
	for i := range sessions {
		if sessions[i].ID == gregID {
			sessions[i].ClaudeSession = claudeSessionID
			break
		}
	}
	SaveSessions(sessions)
}

func AccumulateUsage(gregID string, outputTokens int, cost float64) {
	sessions, err := LoadSessions()
	if err != nil {
		return
	}
	for i := range sessions {
		if sessions[i].ID == gregID {
			sessions[i].OutputTokens += outputTokens
			sessions[i].CostUSD += cost
			break
		}
	}
	SaveSessions(sessions)
}

// ReviveSession moves a finished session from history.json back to sessions.json as active.
func ReviveSession(id string) (*Session, error) {
	history, err := LoadFinishedSessions()
	if err != nil {
		return nil, err
	}
	var revived *Session
	var remaining []Session
	for i := range history {
		if history[i].ID == id {
			s := history[i]
			s.Status = "active"
			s.Ended = ""
			revived = &s
		} else {
			remaining = append(remaining, history[i])
		}
	}
	if revived == nil {
		return nil, nil // not in history, nothing to do
	}
	// Re-create mailbox dir so inbox messages work
	mbDir := filepath.Join(MailboxDir(), id)
	os.MkdirAll(mbDir, 0755)
	inboxPath := filepath.Join(mbDir, "inbox.md")
	if _, err := os.Stat(inboxPath); os.IsNotExist(err) {
		os.WriteFile(inboxPath, []byte{}, 0644)
	}
	// Write updated history
	data, err := json.MarshalIndent(remaining, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(HistoryFile(), data, 0644); err != nil {
		return nil, err
	}
	// Add back to sessions.json
	sessions, _ := LoadSessions()
	sessions = append(sessions, *revived)
	if err := SaveSessions(sessions); err != nil {
		return nil, err
	}
	return revived, nil
}

func FindActiveSession() *Session {
	sessions, err := LoadSessions()
	if err != nil || len(sessions) == 0 {
		return nil
	}
	for i := len(sessions) - 1; i >= 0; i-- {
		if sessions[i].Status == "active" {
			return &sessions[i]
		}
	}
	last := sessions[len(sessions)-1]
	return &last
}
