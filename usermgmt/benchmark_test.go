package usermgmt

import (
	"context"
	"strings"
	"testing"
	"time"
)

func BenchmarkService_Login(b *testing.B) {
	svc, _ := NewService(ServiceConfig{BcryptCost: minBcryptCost})
	ctx := context.Background()
	_, _ = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("bench1"), Email: "bench@test.com", Password: "secret12",
	})

	req := LoginRequest{Email: "bench@test.com", Password: "secret12"}
	b.ResetTimer()
	for range b.N {
		_, _ = svc.Login(ctx, req)
	}
}

func BenchmarkService_Register(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		svc, _ := NewService(ServiceConfig{BcryptCost: minBcryptCost})
		_, _ = svc.Register(context.Background(), RegisterRequest{
			ID:       NewUserID(strings.Repeat("a", 26)),
			Email:    "bench@test.com",
			Password: "secret12",
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
