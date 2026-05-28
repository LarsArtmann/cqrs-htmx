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

func BenchmarkUser_ChangePassword(b *testing.B) {
	u := NewUser(NewUserID("bench"), "bench@test.com", "Bench")
	_ = u.SetPasswordWithCost("oldpass12", minBcryptCost)
	b.ResetTimer()
	for range b.N {
		_, _ = u.ChangePassword("oldpass12", "newpass12", minBcryptCost)
		// Reset for next iteration
		_ = u.SetPasswordWithCost("oldpass12", minBcryptCost)
	}
}

func BenchmarkUser_SetRoles(b *testing.B) {
	u := NewUser(NewUserID("bench"), "bench@test.com", "Bench")
	roles := []Role{RoleAdmin, RoleUser, RoleViewer, RoleOwner}
	b.ResetTimer()
	for range b.N {
		u.SetRoles(roles)
	}
}
