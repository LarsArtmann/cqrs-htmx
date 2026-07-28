package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func newBotTestService(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithConfig(t, ServiceConfig{
		TokenPepper: TokenPepper("test-pepper-32-bytes-long-xxxxxxx"),
	})
}

func TestService_RegisterBot_Success(t *testing.T) {
	t.Parallel()

	svc := newBotTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	botID := NewBotID(id.NewStreamID().String())
	result, err := svc.RegisterBot(ctx, RegisterBotRequest{
		ID:      botID,
		Name:    "deploy-bot",
		OwnerID: NewUserID("owner-1"),
		Scopes:  []string{"deploy", "read"},
	})
	if err != nil {
		t.Fatalf("RegisterBot: %v", err)
	}
	if result.Bot == nil {
		t.Fatal("RegisterBot returned nil bot")
	}
	if result.Token == "" {
		t.Fatal("RegisterBot returned empty token")
	}
	if result.Bot.Name != "deploy-bot" {
		t.Errorf("Name = %q, want %q", result.Bot.Name, "deploy-bot")
	}

	// Verify via read model (read-your-writes)
	fetched, err := svc.GetBot(ctx, botID)
	if err != nil {
		t.Fatalf("GetBot after register: %v", err)
	}
	if fetched.Name != result.Bot.Name {
		t.Errorf("GetBot Name = %q, want %q", fetched.Name, result.Bot.Name)
	}
}

func TestService_RegisterBot_NoPepper(t *testing.T) {
	t.Parallel()

	svc := newTestService(t) // no TokenPepper
	t.Cleanup(svc.Stop)

	_, err := svc.RegisterBot(context.Background(), RegisterBotRequest{
		ID:   NewBotID(id.NewStreamID().String()),
		Name: "bot-no-pepper",
	})
	if err == nil {
		t.Fatal("expected error when TokenPepper not configured, got nil")
	}
}

func TestService_RegisterBot_EmptyName(t *testing.T) {
	t.Parallel()

	svc := newBotTestService(t)
	t.Cleanup(svc.Stop)

	_, err := svc.RegisterBot(context.Background(), RegisterBotRequest{
		ID:   NewBotID(id.NewStreamID().String()),
		Name: "",
	})
	if err == nil {
		t.Fatal("expected error for empty bot name, got nil")
	}
}

func TestService_ResolveBotByToken(t *testing.T) {
	t.Parallel()

	svc := newBotTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	result, err := svc.RegisterBot(ctx, RegisterBotRequest{
		ID:      NewBotID(id.NewStreamID().String()),
		Name:    "resolver-bot",
		OwnerID: NewUserID("owner-1"),
	})
	if err != nil {
		t.Fatalf("RegisterBot: %v", err)
	}

	// Resolve by the raw token (shown once at registration)
	bot, ok := svc.ResolveBotByToken(result.Token)
	if !ok {
		t.Fatal("ResolveBotByToken failed for valid token")
	}
	if bot.Name != "resolver-bot" {
		t.Errorf("resolved bot Name = %q, want %q", bot.Name, "resolver-bot")
	}

	// Invalid token should not resolve
	_, ok = svc.ResolveBotByToken("invalid-token")
	if ok {
		t.Error("ResolveBotByToken should return false for invalid token")
	}
}

func TestService_DeleteBot(t *testing.T) {
	t.Parallel()

	svc := newBotTestService(t)
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	botID := NewBotID(id.NewStreamID().String())
	_, err := svc.RegisterBot(ctx, RegisterBotRequest{
		ID:      botID,
		Name:    "delete-bot",
		OwnerID: NewUserID("owner-1"),
	})
	if err != nil {
		t.Fatalf("RegisterBot: %v", err)
	}

	if err := svc.DeleteBot(ctx, botID, "decommissioned"); err != nil {
		t.Fatalf("DeleteBot: %v", err)
	}

	// Deleted bots are removed from the read model
	_, err = svc.GetBot(ctx, botID)
	if err == nil {
		t.Fatal("expected error getting deleted bot, got nil")
	}
}
