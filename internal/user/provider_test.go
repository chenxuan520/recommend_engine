package user

import (
	"testing"

	"recommend_engine/internal/model"
)

func TestStaticProvider(t *testing.T) {
	// Create a dummy config file for testing
	// For unit tests, it's better to mock the file system or inject data, but StaticProvider reads from file.
	// We will skip file creation and just test the structure if we could, 
	// but here we can't easily mock os.ReadFile without refactoring.
	// So we will just test the logic if we had data.
	
	// Refactor StaticProvider to accept data or reader is better, but for MVP:
	// We will just create a simple integration-like unit test that fails if file missing,
	// or we create a temp file.
	
	// Let's just write a test that we know will fail without config, so we skip it or make it robust.
	// Better: Test the struct logic directly if possible.
	// Since StaticProvider logic is simple map lookups, we can manually construct it.
	
	p := &StaticProvider{
		users: map[string]*model.User{
			"u1": {ID: "u1", Name: "Test User", Token: "t1"},
		},
		tokenIndex: map[string]*model.User{
			"t1": {ID: "u1", Name: "Test User", Token: "t1"},
		},
	}

	// Test GetUser
	u, err := p.GetUser("u1")
	if err != nil {
		t.Errorf("GetUser failed: %v", err)
	}
	if u.Name != "Test User" {
		t.Errorf("Expected 'Test User', got %s", u.Name)
	}

	// Test GetUserByToken
	u2, err := p.GetUserByToken("t1")
	if err != nil {
		t.Errorf("GetUserByToken failed: %v", err)
	}
	if u2.ID != "u1" {
		t.Errorf("Expected u1, got %s", u2.ID)
	}

	// Test NotFound
	_, err = p.GetUser("u2")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}
