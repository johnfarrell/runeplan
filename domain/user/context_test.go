package user_test

import (
	"context"
	"testing"

	"github.com/johnfarrell/runeplan/domain/user"
)

func TestSetGetUser(t *testing.T) {
	u := user.User{ID: "abc123"}
	ctx := user.SetUser(context.Background(), u)
	got, ok := user.GetUser(ctx)
	if !ok {
		t.Fatal("expected user in context")
	}
	if got.ID != "abc123" {
		t.Errorf("got ID %q, want %q", got.ID, "abc123")
	}
}

func TestGetUser_Missing(t *testing.T) {
	_, ok := user.GetUser(context.Background())
	if ok {
		t.Fatal("expected no user in empty context")
	}
}
