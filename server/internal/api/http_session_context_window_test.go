package api

import (
	"testing"

	agenttypes "mindfs/server/internal/agent/types"
	"mindfs/server/internal/session"
)

func TestSessionContextWindowPayloadFallsBackToPersistedExchange(t *testing.T) {
	persisted := &agenttypes.ContextWindow{
		TotalTokens:        1200,
		ModelContextWindow: 32000,
	}
	sess := &session.Session{
		Exchanges: []session.Exchange{
			{Role: "user", Content: "hi"},
			{Role: "agent", Content: "hello", ContextWindow: persisted},
		},
	}

	got := sessionContextWindowPayload(sess, agenttypes.ContextWindow{})
	if got.TotalTokens != persisted.TotalTokens || got.ModelContextWindow != persisted.ModelContextWindow {
		t.Fatalf("unexpected fallback context window: %#v", got)
	}
}

func TestSessionContextWindowPayloadPrefersLiveContext(t *testing.T) {
	live := agenttypes.ContextWindow{
		TotalTokens:        2400,
		ModelContextWindow: 64000,
	}
	persisted := &agenttypes.ContextWindow{
		TotalTokens:        1200,
		ModelContextWindow: 32000,
	}
	sess := &session.Session{
		Exchanges: []session.Exchange{
			{Role: "agent", Content: "hello", ContextWindow: persisted},
		},
	}

	got := sessionContextWindowPayload(sess, live)
	if got.TotalTokens != live.TotalTokens || got.ModelContextWindow != live.ModelContextWindow {
		t.Fatalf("unexpected live context window: %#v", got)
	}
}

func TestSessionListResponseIncludesPersistedContextWindow(t *testing.T) {
	persisted := &agenttypes.ContextWindow{
		TotalTokens:        900,
		ModelContextWindow: 128000,
	}
	payload := sessionListResponse(&session.Session{
		Key:  "s1",
		Type: session.TypeChat,
		Name: "demo",
		Exchanges: []session.Exchange{
			{Role: "agent", Content: "hello", ContextWindow: persisted},
		},
	})

	raw, ok := payload["context_window"]
	if !ok {
		t.Fatal("expected context_window in payload")
	}
	got, ok := raw.(agenttypes.ContextWindow)
	if !ok {
		t.Fatalf("unexpected context_window type: %T", raw)
	}
	if got.TotalTokens != persisted.TotalTokens || got.ModelContextWindow != persisted.ModelContextWindow {
		t.Fatalf("unexpected context_window payload: %#v", got)
	}
}
