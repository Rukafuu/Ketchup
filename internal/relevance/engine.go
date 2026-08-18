package relevance

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/ketchup-ai/ketchup/internal/signals"
)

// Contribution represents a single rule's contribution to a relevance score
type Contribution struct {
	Rule    string   `json:"rule"`
	Delta   int      `json:"delta"`
	Reason  string   `json:"reason"`
	Matches []string `json:"matches,omitempty"`
}

// RelevanceSignal represents a computed relevance signal
type RelevanceSignal struct {
	// Score is the relevance score (0-100)
	Score int `json:"score"`

	// Reasons explains why this event is relevant
	Reasons []string `json:"reasons"`

	// Severity indicates the severity (CRITICAL, HIGH, MEDIUM, LOW, IGNORED)
	Severity string `json:"severity"`

	// Contributions lists individual rule contributions
	Contributions []Contribution `json:"contributions"`

	// Threshold used for the decision
	Threshold int `json:"threshold"`

	// Decision is RELEVANT or IGNORED
	Decision string `json:"decision"`
}

// RelevantChange is an event with its computed relevance
type RelevantChange struct {
	Event        signals.NormalizedEvent `json:"event"`
	Signal       RelevanceSignal         `json:"signal"`
	Ignored      bool                    `json:"ignored"` // true if filtered by low relevance
	Contributions []Contribution         `json:"contributions,omitempty"`
}

// Engine computes relevance of events based on context
type Engine struct {
	// CurrentBranch is the developer's current branch
	CurrentBranch string

	// CurrentFiles are files currently open/edited
	CurrentFiles []string

	// RecentFiles are files recently modified locally
	RecentFiles []string

	// Developer is the name of the current developer
	Developer string

	// Threshold for considering an event relevant (default 20)
	Threshold int
}

// NewEngine creates a new relevance engine
func NewEngine() *Engine {
	return &Engine{
		Threshold: 20,
	}
}

// ComputeRelevance computes relevance for a set of events (legacy method)
func (e *Engine) ComputeRelevance(events []signals.NormalizedEvent) []RelevantChange {
	var changes []RelevantChange

	for _, event := range events {
		signal := e.computeEventRelevance(event)
		change := RelevantChange{
			Event:   event,
			Signal:  signal,
			Ignored: signal.Score < e.Threshold,
		}
		changes = append(changes, change)
	}

	return changes
}

// ComputeRelevanceWithContributions computes relevance with structured contributions
func (e *Engine) ComputeRelevanceWithContributions(events []signals.NormalizedEvent) []RelevantChange {
	var changes []RelevantChange

	for _, event := range events {
		signal, contributions := e.computeEventRelevanceWithContributions(event)
		change := RelevantChange{
			Event:        event,
			Signal:       signal,
			Ignored:      signal.Decision == "IGNORED",
			Contributions: contributions,
		}
		changes = append(changes, change)
	}

	// Sort by score descending, then by timestamp descending
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Signal.Score != changes[j].Signal.Score {
			return changes[i].Signal.Score > changes[j].Signal.Score
		}
		return changes[i].Event.Timestamp.After(changes[j].Event.Timestamp)
	})

	return changes
}

// computeEventRelevance computes relevance for a single event (legacy)
func (e *Engine) computeEventRelevance(event signals.NormalizedEvent) RelevanceSignal {
	signal := RelevanceSignal{
		Score:   0,
		Reasons: []string{},
	}

	// 1. Overlap with current/recent files (high weight)
	for _, currentFile := range e.CurrentFiles {
		if e.fileInEvent(currentFile, event) {
			signal.Score += 40
			signal.Reasons = append(signal.Reasons,
				"This commit changed a file you currently have open")
		}
	}

	for _, recentFile := range e.RecentFiles {
		if e.fileInEvent(recentFile, event) {
			signal.Score += 30
			signal.Reasons = append(signal.Reasons,
				"This commit changed a file you recently edited")
		}
	}

	// 2. Same directory/module (medium weight)
	for _, currentFile := range e.CurrentFiles {
		dir := filepath.Dir(currentFile)
		if e.dirInEvent(dir, event) {
			signal.Score += 20
			signal.Reasons = append(signal.Reasons,
				"This commit changed files in the same directory as your work")
		}
	}

	// 3. Dependency/config files (high weight)
	for _, file := range event.Files {
		if isCriticalFile(file) {
			signal.Score += 25
			signal.Reasons = append(signal.Reasons,
				"This commit changed a critical configuration or dependency file")
		}
	}

	// 4. Migration files (high weight)
	for _, file := range event.Files {
		if isMigrationFile(file) {
			signal.Score += 30
			signal.Reasons = append(signal.Reasons,
				"This commit changed a database migration file")
		}
	}

	// 5. Merge commits (additional weight)
	if event.Type == "merge" {
		signal.Score += 15
		signal.Reasons = append(signal.Reasons,
			"This is a merge commit")
	}

	// 6. Relevant author (if known developer)
	if e.Developer != "" && event.Actor == e.Developer {
		// Own commits have lower priority
		signal.Score -= 10
		signal.Reasons = append(signal.Reasons,
			"This is your own commit")
	}

	// Normalize score to 0-100
	if signal.Score > 100 {
		signal.Score = 100
	}
	if signal.Score < 0 {
		signal.Score = 0
	}

	// Determine severity
	signal.Severity = e.computeSeverity(signal.Score)

	return signal
}

// computeEventRelevanceWithContributions computes relevance with structured contributions
func (e *Engine) computeEventRelevanceWithContributions(event signals.NormalizedEvent) (RelevanceSignal, []Contribution) {
	signal := RelevanceSignal{
		Score:       0,
		Reasons:     []string{},
		Threshold:   e.Threshold,
		Contributions: []Contribution{},
	}
	var contributions []Contribution

	threshold := e.Threshold
	if threshold <= 0 {
		threshold = 20
	}

	// Track which rules have been applied to avoid double-counting
	appliedRules := make(map[string]bool)

	// 1. Current file overlap (max 40 points)
	for _, currentFile := range e.CurrentFiles {
		matchedFiles := e.findMatchingFilesInEvent(currentFile, event)
		if len(matchedFiles) > 0 && !appliedRules["current_file"] {
			contrib := Contribution{
				Rule:    "current_file",
				Delta:   40,
				Reason:  "This commit changed a file you currently have open",
				Matches: normalizePaths(matchedFiles),
			}
			contributions = append(contributions, contrib)
			signal.Score += 40
			signal.Reasons = append(signal.Reasons, contrib.Reason)
			appliedRules["current_file"] = true
		}
	}

	// 2. Recent file overlap (max 30 points)
	for _, recentFile := range e.RecentFiles {
		matchedFiles := e.findMatchingFilesInEvent(recentFile, event)
		if len(matchedFiles) > 0 && !appliedRules["recent_file"] {
			contrib := Contribution{
				Rule:    "recent_file",
				Delta:   30,
				Reason:  "This commit changed a file you recently edited",
				Matches: normalizePaths(matchedFiles),
			}
			contributions = append(contributions, contrib)
			signal.Score += 30
			signal.Reasons = append(signal.Reasons, contrib.Reason)
			appliedRules["recent_file"] = true
		}
	}

	// 3. Same module/directory (max 20 points)
	var matchedDirs []string
	for _, currentFile := range e.CurrentFiles {
		dir := filepath.Dir(currentFile)
		if e.dirInEvent(dir, event) {
			matchedDirs = append(matchedDirs, dir)
		}
	}
	if len(matchedDirs) > 0 && !appliedRules["same_module"] {
		contrib := Contribution{
			Rule:    "same_module",
			Delta:   20,
			Reason:  "This commit changed files in the same directory as your work",
			Matches: matchedDirs,
		}
		contributions = append(contributions, contrib)
		signal.Score += 20
		signal.Reasons = append(signal.Reasons, contrib.Reason)
		appliedRules["same_module"] = true
	}

	// 4. Critical dependency/config files (max 25 points)
	var criticalFiles []string
	for _, file := range event.Files {
		if isCriticalFile(file) {
			criticalFiles = append(criticalFiles, file)
		}
	}
	if len(criticalFiles) > 0 && !appliedRules["critical_file"] {
		contrib := Contribution{
			Rule:    "critical_file",
			Delta:   25,
			Reason:  "This commit changed a critical configuration or dependency file",
			Matches: normalizePaths(criticalFiles),
		}
		contributions = append(contributions, contrib)
		signal.Score += 25
		signal.Reasons = append(signal.Reasons, contrib.Reason)
		appliedRules["critical_file"] = true
	}

	// 5. Migration files (max 30 points)
	var migrationFiles []string
	for _, file := range event.Files {
		if isMigrationFile(file) {
			migrationFiles = append(migrationFiles, file)
		}
	}
	if len(migrationFiles) > 0 && !appliedRules["migration_file"] {
		contrib := Contribution{
			Rule:    "migration_file",
			Delta:   30,
			Reason:  "This commit changed a database migration file",
			Matches: normalizePaths(migrationFiles),
		}
		contributions = append(contributions, contrib)
		signal.Score += 30
		signal.Reasons = append(signal.Reasons, contrib.Reason)
		appliedRules["migration_file"] = true
	}

	// 6. Merge commits (max 15 points)
	if event.Type == "merge" && !appliedRules["merge_commit"] {
		contrib := Contribution{
			Rule:   "merge_commit",
			Delta:  15,
			Reason: "This is a merge commit",
		}
		contributions = append(contributions, contrib)
		signal.Score += 15
		signal.Reasons = append(signal.Reasons, contrib.Reason)
		appliedRules["merge_commit"] = true
	}

	// 7. Own commit (max -10 points)
	if e.Developer != "" && event.Actor == e.Developer && !appliedRules["own_commit"] {
		contrib := Contribution{
			Rule:   "own_commit",
			Delta:  -10,
			Reason: "This is your own commit (lower priority)",
		}
		contributions = append(contributions, contrib)
		signal.Score -= 10
		signal.Reasons = append(signal.Reasons, contrib.Reason)
		appliedRules["own_commit"] = true
	}

	// Normalize score to 0-100
	if signal.Score > 100 {
		signal.Score = 100
	}
	if signal.Score < 0 {
		signal.Score = 0
	}

	// Determine severity and decision
	signal.Severity = e.computeSeverity(signal.Score)
	if signal.Score >= threshold {
		signal.Decision = "RELEVANT"
	} else {
		signal.Decision = "IGNORED"
	}

	signal.Contributions = contributions

	return signal, contributions
}

// fileInEvent verifica se um arquivo está na lista de arquivos do evento
func (e *Engine) fileInEvent(file string, event signals.NormalizedEvent) bool {
	for _, f := range event.Files {
		if f == file {
			return true
		}
	}
	return false
}

// dirInEvent verifica se algum arquivo do evento está em um diretório
func (e *Engine) dirInEvent(dir string, event signals.NormalizedEvent) bool {
	for _, f := range event.Files {
		if strings.HasPrefix(f, dir+"/") || strings.HasPrefix(f, dir+"\\") {
			return true
		}
	}
	return false
}

// isCriticalFile verifica se é um arquivo crítico
func isCriticalFile(file string) bool {
	criticalPatterns := []string{
		"go.mod", "go.sum",
		"package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock",
		"requirements.txt", "Pipfile.lock", "poetry.lock",
		".env.example", ".env.template",
		"Dockerfile", "docker-compose.yml",
		"Makefile", "Taskfile.yml",
		".github/", ".gitlab-ci.yml",
		"tsconfig.json", "webpack.config.js",
	}

	for _, pattern := range criticalPatterns {
		if file == pattern || strings.Contains(file, "/"+pattern) {
			return true
		}
	}

	return false
}

// isMigrationFile verifica se é um arquivo de migração
func isMigrationFile(file string) bool {
	migrationPatterns := []string{
		"/migrations/", "/migration/",
		"_migration.", "_migrate.",
		".up.sql", ".down.sql",
	}

	for _, pattern := range migrationPatterns {
		if strings.Contains(file, pattern) {
			return true
		}
	}

	return false
}

// computeSeverity determina severidade baseada no score
func (e *Engine) computeSeverity(score int) string {
	switch {
	case score >= 70:
		return "CRITICAL"
	case score >= 50:
		return "HIGH"
	case score >= 30:
		return "MEDIUM"
	case score >= 20:
		return "LOW"
	default:
		return "IGNORED"
	}
}

// findMatchingFilesInEvent finds files in the event that match a given path
func (e *Engine) findMatchingFilesInEvent(path string, event signals.NormalizedEvent) []string {
	var matches []string
	for _, f := range event.Files {
		if normalizePath(f) == normalizePath(path) {
			matches = append(matches, f)
		}
	}
	return matches
}

// normalizePaths normalizes a slice of paths for cross-platform consistency
func normalizePaths(paths []string) []string {
	normalized := make([]string, len(paths))
	for i, p := range paths {
		normalized[i] = normalizePath(p)
	}
	return normalized
}

// normalizePath normalizes a path for cross-platform consistency
func normalizePath(path string) string {
	// Convert backslashes to forward slashes for consistency
	path = strings.ReplaceAll(path, "\\", "/")
	// Clean the path
	path = filepath.Clean(path)
	return path
}
