package signals

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const maxListedFiles = 3

// SummarizeChangedFiles builds a short human-readable summary from modified paths.
func SummarizeChangedFiles(files []string) string {
	if len(files) == 0 {
		return ""
	}

	normalized := normalizeFileList(files)
	if len(normalized) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %d file(s): ", len(normalized)))

	groups := groupFilesByArea(normalized)
	groupLabels := sortedGroupKeys(groups)

	for i, label := range groupLabels {
		if i > 0 {
			sb.WriteString("; ")
		}
		names := fileBasenames(groups[label], maxListedFiles)
		sb.WriteString(fmt.Sprintf("%s (%s)", label, strings.Join(names, ", ")))
		remaining := len(groups[label]) - len(names)
		if remaining > 0 {
			sb.WriteString(fmt.Sprintf(" +%d", remaining))
		}
	}

	return sb.String()
}

func normalizeFileList(files []string) []string {
	seen := make(map[string]bool)
	var normalized []string
	for _, file := range files {
		file = strings.TrimSpace(filepath.ToSlash(file))
		if file == "" || seen[file] {
			continue
		}
		seen[file] = true
		normalized = append(normalized, file)
	}
	sort.Strings(normalized)
	return normalized
}

func groupFilesByArea(files []string) map[string][]string {
	groups := make(map[string][]string)
	for _, file := range files {
		area := classifyFileArea(file)
		groups[area] = append(groups[area], file)
	}
	return groups
}

func classifyFileArea(path string) string {
	lower := strings.ToLower(path)

	switch {
	case strings.HasPrefix(lower, "extension/"):
		return "extension"
	case strings.HasPrefix(lower, "cmd/"):
		return "CLI"
	case strings.HasPrefix(lower, "internal/"):
		return "core"
	case strings.HasPrefix(lower, "docs/"):
		return "docs"
	case strings.HasPrefix(lower, "scripts/"):
		return "scripts"
	case strings.HasPrefix(lower, "assets/"):
		return "assets"
	case strings.HasPrefix(lower, ".github/"):
		return "CI"
	case strings.HasSuffix(lower, ".md"):
		return "docs"
	case isDependencyOrConfigFile(lower):
		return "config/deps"
	default:
		if idx := strings.Index(lower, "/"); idx > 0 {
			return lower[:idx]
		}
		return "root"
	}
}

func isDependencyOrConfigFile(path string) bool {
	base := filepath.Base(path)
	switch base {
	case "go.mod", "go.sum", "package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock",
		".ketchup.yaml", ".ketchup.yml", "docker-compose.yml", "Dockerfile":
		return true
	}
	return strings.HasPrefix(base, ".env")
}

func fileBasenames(files []string, limit int) []string {
	if limit <= 0 || len(files) == 0 {
		return nil
	}
	if len(files) < limit {
		limit = len(files)
	}
	names := make([]string, 0, limit)
	for _, file := range files[:limit] {
		names = append(names, filepath.Base(file))
	}
	return names
}

func sortedGroupKeys(groups map[string][]string) []string {
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(groups[keys[i]]) != len(groups[keys[j]]) {
			return len(groups[keys[i]]) > len(groups[keys[j]])
		}
		return keys[i] < keys[j]
	})
	return keys
}
