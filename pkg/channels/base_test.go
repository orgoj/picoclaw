package channels

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestBaseChannelIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		allowList []string
		senderID  string
		want      bool
	}{
		{
			name:      "empty allowlist allows all",
			allowList: nil,
			senderID:  "anyone",
			want:      true,
		},
		{
			name:      "compound sender matches numeric allowlist",
			allowList: []string{"123456"},
			senderID:  "123456|alice",
			want:      true,
		},
		{
			name:      "compound sender matches username allowlist",
			allowList: []string{"@alice"},
			senderID:  "123456|alice",
			want:      true,
		},
		{
			name:      "numeric sender matches legacy compound allowlist",
			allowList: []string{"123456|alice"},
			senderID:  "123456",
			want:      true,
		},
		{
			name:      "non matching sender is denied",
			allowList: []string{"123456"},
			senderID:  "654321|bob",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := NewBaseChannel("test", nil, nil, tt.allowList)
			if got := ch.IsAllowed(tt.senderID); got != tt.want {
				t.Fatalf("IsAllowed(%q) = %v, want %v", tt.senderID, got, tt.want)
			}
		})
	}
}

func TestBaseChannelHandleMessage_AddsPeerRoutingMetadata(t *testing.T) {
	mb := bus.NewMessageBus()
	ch := NewBaseChannel("line", nil, mb, nil)

	ch.HandleMessage("u1", "chat-42", "hello", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := mb.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message")
	}
	if got := msg.Metadata["peer_kind"]; got != "chat" {
		t.Fatalf("peer_kind=%q want=chat", got)
	}
	if got := msg.Metadata["peer_id"]; got != "chat-42" {
		t.Fatalf("peer_id=%q want=chat-42", got)
	}
}

func TestBaseChannelHandleMessage_RespectsProvidedPeerRoutingMetadata(t *testing.T) {
	mb := bus.NewMessageBus()
	ch := NewBaseChannel("discord", nil, mb, nil)

	ch.HandleMessage("u1", "chat-42", "hello", nil, map[string]string{
		"peer_kind": "user",
		"peer_id":   "alice",
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := mb.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message")
	}
	if got := msg.Metadata["peer_kind"]; got != "user" {
		t.Fatalf("peer_kind=%q want=user", got)
	}
	if got := msg.Metadata["peer_id"]; got != "alice" {
		t.Fatalf("peer_id=%q want=alice", got)
	}
}
