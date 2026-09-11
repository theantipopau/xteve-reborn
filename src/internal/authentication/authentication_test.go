package authentication

import (
	"path/filepath"
	"testing"
)

func TestCreateDefaultUserAndAuthenticate(t *testing.T) {
	dir := t.TempDir()
	if err := Init(filepath.Join(dir, "config"), 60); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := CreateDefaultUser("admin", "correct horse battery staple"); err != nil {
		t.Fatalf("CreateDefaultUser() error = %v", err)
	}

	token, err := UserAuthentication("admin", "correct horse battery staple")
	if err != nil {
		t.Fatalf("UserAuthentication() with correct credentials error = %v", err)
	}
	if token == "" {
		t.Fatal("UserAuthentication() returned an empty token for valid credentials")
	}

	if _, err := UserAuthentication("admin", "wrong password"); err == nil {
		t.Fatal("UserAuthentication() should fail with the wrong password")
	}

	if _, err := UserAuthentication("nobody", "correct horse battery staple"); err == nil {
		t.Fatal("UserAuthentication() should fail for an unknown username")
	}

	newToken, err := CheckTheValidityOfTheToken(token)
	if err != nil {
		t.Fatalf("CheckTheValidityOfTheToken() error = %v", err)
	}
	if newToken == "" {
		t.Fatal("CheckTheValidityOfTheToken() returned an empty token")
	}

	userID, err := GetUserID(newToken)
	if err != nil {
		t.Fatalf("GetUserID() error = %v", err)
	}
	if userID == "" {
		t.Fatal("GetUserID() returned an empty user id")
	}
}

func TestHashSecretUsesSaltAndIsDeterministic(t *testing.T) {
	a := hashSecret("same-password", "salt-one")
	b := hashSecret("same-password", "salt-two")
	if a == b {
		t.Fatal("hashSecret() produced identical output for two different salts - the salt isn't being used")
	}

	c := hashSecret("same-password", "salt-one")
	if a != c {
		t.Fatal("hashSecret() is not deterministic for the same secret+salt pair")
	}
}
