package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

var botTestAggID = id.NewAggregateID() //nolint:gochecknoglobals // test fixture

func makeBotEvent(t *testing.T, eventType event.Type, version event.Version, payload any) event.Event {
	t.Helper()
	payloadBytes, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal payload for %s: %v", eventType, err)
	}
	evt, err := event.NewEvent(eventType, botTestAggID, aggregateTypeBot, version, payloadBytes)
	if err != nil {
		t.Fatalf("makeBotEvent %s: %v", eventType, err)
	}
	return evt
}

func TestFoldBot_Registered(t *testing.T) {
	state, err := foldBot(BotState{}, makeBotEvent(t, eventBotRegistered, 1, BotRegisteredPayload{
		Name:      "ci-bot",
		OwnerID:   NewUserID("user-123"),
		TokenHash: []byte{1, 2, 3},
		Scopes:    []string{"read", "write"},
	}))
	if err != nil {
		t.Fatalf("foldBot: %v", err)
	}
	if state.Name != "ci-bot" {
		t.Errorf("Name = %q", state.Name)
	}
	if state.OwnerID.Get().String() != NewUserID("user-123").Get().String() {
		t.Errorf("OwnerID = %s", state.OwnerID.Get().String())
	}
	if len(state.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(state.Scopes))
	}
	if !state.Exists() {
		t.Error("bot should exist after registration")
	}
}

func TestFoldBot_Deleted(t *testing.T) {
	state := BotState{Name: "ci-bot", OwnerID: NewUserID("user-123")}
	deletedEvt, err := event.NewEvent(
		eventBotDeleted, botTestAggID, aggregateTypeBot, 2,
		mustMarshalPayload(t, BotDeletedPayload{Reason: "rotated"}),
	)
	if err != nil {
		t.Fatalf("create deleted event: %v", err)
	}
	deletedEvt, err = event.MarkTombstone(deletedEvt)
	if err != nil {
		t.Fatalf("mark tombstone: %v", err)
	}
	state, err = foldBot(state, deletedEvt)
	if err != nil {
		t.Fatalf("foldBot: %v", err)
	}
	if !state.Deleted {
		t.Error("bot should be deleted")
	}
	if state.Exists() {
		t.Error("deleted bot should not exist")
	}
}

func TestFoldBot_UnknownEvent(t *testing.T) {
	unknownEvt, err := event.NewEvent(
		event.Type("FutureBotEvent"), botTestAggID, aggregateTypeBot, 1,
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("create unknown event: %v", err)
	}
	_, err = foldBot(BotState{}, unknownEvt)
	if err == nil {
		t.Error("expected error for unknown event")
	}
}

func TestDecideRegisterBot_AlreadyExists(t *testing.T) {
	state := BotState{Name: "existing"}
	decide := decideRegisterBot(botTestAggID, "new", NewUserID("owner"), []byte{1}, nil)
	_, err := decide(state, 1)
	if err == nil {
		t.Error("expected conflict for already-existing bot")
	}
}

func TestDecideRegisterBot_EmptyName(t *testing.T) {
	decide := decideRegisterBot(botTestAggID, "", NewUserID("owner"), []byte{1}, nil)
	_, err := decide(BotState{}, 1)
	if err == nil {
		t.Error("expected rejection for empty name")
	}
}

func TestDecideRegisterBot_EmptyOwner(t *testing.T) {
	decide := decideRegisterBot(botTestAggID, "bot", UserID{}, []byte{1}, nil)
	_, err := decide(BotState{}, 1)
	if err == nil {
		t.Error("expected rejection for empty owner")
	}
}

func TestDecideRegisterBot_EmptyTokenHash(t *testing.T) {
	decide := decideRegisterBot(botTestAggID, "bot", NewUserID("owner"), nil, nil)
	_, err := decide(BotState{}, 1)
	if err == nil {
		t.Error("expected rejection for empty token hash")
	}
}

func TestDecideDeleteBot_NotFound(t *testing.T) {
	decide := decideDeleteBot(botTestAggID, "test")
	_, err := decide(BotState{}, 1)
	if err == nil {
		t.Error("expected rejection for non-existent bot")
	}
}

func TestDecideDeleteBot_AlreadyDeleted(t *testing.T) {
	state := BotState{Name: "bot", Deleted: true}
	decide := decideDeleteBot(botTestAggID, "test")
	_, err := decide(state, 1)
	if err == nil {
		t.Error("expected rejection for already-deleted bot")
	}
}

func TestBotReadModel_RegisterAndFind(t *testing.T) {
	rm := NewBotReadModel()
	ctx := t.Context()

	evt := makeBotEvent(t, eventBotRegistered, 1, BotRegisteredPayload{
		Name:      "ci-bot",
		OwnerID:   NewUserID("user-123"),
		TokenHash: []byte{0xAA, 0xBB},
		Scopes:    []string{"read"},
	})
	if err := rm.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	bot, ok := rm.FindByID(botTestAggID)
	if !ok {
		t.Fatal("bot not found by ID")
	}
	if bot.Name != "ci-bot" {
		t.Errorf("Name = %q", bot.Name)
	}

	bot2, ok := rm.FindByTokenHash([]byte{0xAA, 0xBB})
	if !ok {
		t.Fatal("bot not found by token hash")
	}
	if bot2.Name != "ci-bot" {
		t.Errorf("Name = %q", bot2.Name)
	}
}

func TestBotReadModel_DeleteRemovesFromIndexes(t *testing.T) {
	rm := NewBotReadModel()
	ctx := t.Context()

	if err := rm.Handle(ctx, makeBotEvent(t, eventBotRegistered, 1, BotRegisteredPayload{
		Name: "ci-bot", OwnerID: NewUserID("u1"), TokenHash: []byte{0xAA},
	})); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	deletedEvt, err := event.NewEvent(
		eventBotDeleted, botTestAggID, aggregateTypeBot, 2,
		mustMarshalPayload(t, BotDeletedPayload{Reason: "test"}),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	deletedEvt, _ = event.MarkTombstone(deletedEvt)

	if err := rm.Handle(ctx, deletedEvt); err != nil {
		t.Fatalf("Handle delete: %v", err)
	}

	if _, ok := rm.FindByID(botTestAggID); ok {
		t.Error("bot should be removed from ID index")
	}
	if _, ok := rm.FindByTokenHash([]byte{0xAA}); ok {
		t.Error("bot should be removed from token hash index")
	}
}
