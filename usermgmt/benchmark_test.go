package usermgmt

import (
	"context"
	"strings"
	"testing"
	"time"
)

func BenchmarkService_Register(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		svc, _ := NewService(ServiceConfig{})
		_, _ = svc.Register(context.Background(), RegisterRequest{
			ID:    NewUserID(strings.Repeat("a", 26)),
			Email: "bench@test.com",
		})
	}
}

func BenchmarkSession_TokenMatches(b *testing.B) {
	s, _ := NewSession(NewUserID("bench"), time.Hour)
	b.ResetTimer()
	for range b.N {
		_ = s.TokenMatches(s.Token)
	}
}
