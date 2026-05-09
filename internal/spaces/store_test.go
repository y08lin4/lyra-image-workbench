package spaces

import "testing"

const testPassword = "R7!Blue#Vault$2026"

func TestCreateOrOpenByPasswordCreatesStableSpace(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	first, err := store.CreateOrOpenByPassword(testPassword)
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword(first) error = %v", err)
	}
	if !first.Created {
		t.Fatal("first open should create the space")
	}
	if first.Token == "" || first.TokenPreview == "" {
		t.Fatalf("session token fields should be populated: %+v", first)
	}

	second, err := store.CreateOrOpenByPassword(testPassword)
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword(second) error = %v", err)
	}
	if second.Created {
		t.Fatal("second open should reuse the existing space")
	}
	if second.Token != first.Token {
		t.Fatalf("token changed: %q != %q", second.Token, first.Token)
	}

	opened, err := store.OpenByToken(first.Token)
	if err != nil {
		t.Fatalf("OpenByToken() error = %v", err)
	}
	if opened.Space.ID != first.Space.ID {
		t.Fatalf("opened wrong space: %+v", opened)
	}
}

func TestPasswordValidationRejectsWeakInput(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected weak password to be rejected")
	}
	if _, err := NormalizeToken("not-a-token"); err == nil {
		t.Fatal("expected invalid token to be rejected")
	}
}
