package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session representa o estado da última sessão do Ketchup em um workspace
type Session struct {
	// ID é um identificador único da sessão
	ID string `json:"id"`
	
	// RepositoryRoot é o caminho absoluto do repositório
	RepositoryRoot string `json:"repository_root"`
	
	// LastActivity é quando o usuário usou o Ketchup pela última vez
	LastActivity time.Time `json:"last_activity"`
	
	// HeadCommit é o commit HEAD na última sessão
	HeadCommit string `json:"head_commit,omitempty"`
	
	// Branch é o branch ativo na última sessão
	Branch string `json:"branch,omitempty"`
	
	// RecentFiles são arquivos recentemente editados (opcional, para relevância futura)
	RecentFiles []string `json:"recent_files,omitempty"`
}

// Store persiste a sessão atual no diretório .ketchup
type Store struct {
	ketchupDir string
}

// NewStore cria uma nova Store para um repositório
func NewStore(repoRoot string) *Store {
	return &Store{
		ketchupDir: filepath.Join(repoRoot, ".ketchup"),
	}
}

// Load carrega a última sessão salva
func (s *Store) Load() (*Session, error) {
	sessionPath := filepath.Join(s.ketchupDir, "session.json")
	
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSession
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}
	
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("invalid session file: %w", err)
	}
	
	return &session, nil
}

// Save persiste a sessão atual
func (s *Store) Save(session *Session) error {
	// Cria diretório .ketchup se não existir
	if err := os.MkdirAll(s.ketchupDir, 0755); err != nil {
		return fmt.Errorf("failed to create .ketchup directory: %w", err)
	}
	
	// Cria .gitignore dentro de .ketchup para ignorar session.json
	gitignorePath := filepath.Join(s.ketchupDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitignorePath, []byte("session.json\n"), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
	}
	
	sessionPath := filepath.Join(s.ketchupDir, "session.json")
	
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	
	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}
	
	return nil
}

// UpdateOrCreate atualiza a sessão existente ou cria uma nova
func (s *Store) UpdateOrCreate(repoRoot, headCommit, branch string) (*Session, error) {
	session, err := s.Load()
	if err == ErrNoSession {
		// Cria nova sessão
		session = &Session{
			ID:             fmt.Sprintf("session-%d", time.Now().Unix()),
			RepositoryRoot: repoRoot,
			LastActivity:   time.Now(),
			HeadCommit:     headCommit,
			Branch:         branch,
		}
	} else if err != nil {
		return nil, err
	} else {
		// Atualiza sessão existente
		session.LastActivity = time.Now()
		session.HeadCommit = headCommit
		session.Branch = branch
	}
	
	if err := s.Save(session); err != nil {
		return nil, err
	}
	
	return session, nil
}

// ErrNoSession é retornado quando não há sessão salva
var ErrNoSession = &noSessionError{}

type noSessionError struct{}

func (e *noSessionError) Error() string {
	return "no previous session found"
}
