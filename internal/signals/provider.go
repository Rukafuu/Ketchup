package signals

import (
	"context"
	"time"
)

// LastSessionInfo contém informações sobre a última sessão
type LastSessionInfo struct {
	Timestamp  time.Time
	HeadCommit string
	Branch     string
}

// Provider é a interface para provedores de sinais
type Provider interface {
	// Name retorna o nome do provider
	Name() string

	// FetchEvents busca eventos normalizados desde uma sessão anterior
	FetchEvents(ctx context.Context, root string, since LastSessionInfo) ([]NormalizedEvent, error)
}
