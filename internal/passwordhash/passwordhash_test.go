package passwordhash

import "testing"

func TestNewVerifyAndLegacyUpgradeSignal(t *testing.T) {
	password := "R7!Blue#Vault$2026"
	salt, encoded, err := New(password)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if salt == "" || encoded == "" {
		t.Fatalf("New() returned empty values salt=%q encoded=%q", salt, encoded)
	}
	ok, upgrade := Verify(salt, encoded, password)
	if !ok || upgrade {
		t.Fatalf("Verify(new) ok=%v upgrade=%v", ok, upgrade)
	}
	ok, upgrade = Verify("legacy-salt", LegacyHash("legacy-salt", password), password)
	if !ok || !upgrade {
		t.Fatalf("Verify(legacy) ok=%v upgrade=%v", ok, upgrade)
	}
	ok, _ = Verify(salt, encoded, "Wrong!Password123")
	if ok {
		t.Fatal("Verify() accepted wrong password")
	}
}
