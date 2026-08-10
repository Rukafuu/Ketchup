package signals

import "time"

// Event representa um evento normalizado de qualquer provedor de sinais
type Event struct {
	// ID é um identificador único do evento
	ID string `json:"id"`

	// Source é a origem do evento (ex: "git", "jira", "slack")
	Source string `json:"source"`

	// Type é o tipo do evento (ex: "commit", "merge", "ticket_updated")
	Type string `json:"type"`

	// Timestamp é quando o evento ocorreu
	Timestamp time.Time `json:"timestamp"`

	// Actor é quem realizou a ação (quando disponível)
	Actor string `json:"actor,omitempty"`

	// Title é um resumo curto do evento
	Title string `json:"title"`

	// Description é uma descrição mais detalhada
	Description string `json:"description,omitempty"`

	// Files são arquivos afetados pelo evento
	Files []string `json:"files,omitempty"`

	// Modules são módulos/pacotes afetados
	Modules []string `json:"modules,omitempty"`

	// References são referências externas (tickets, PRs, etc.)
	References []string `json:"references,omitempty"`

	// Metadata contém dados específicos do provider
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NormalizedEvent é um alias para Event (para clareza semântica)
type NormalizedEvent = Event
