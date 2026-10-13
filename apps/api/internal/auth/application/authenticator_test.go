package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

func TestAuthenticateValidAPIKey(t *testing.T) {
	token := "if_test_valid_credential_123456789"
	sum := sha256.Sum256([]byte(token))
	repository := &apiKeyRepositoryStub{key: APIKey{ID: "key_1", ProjectID: "prj_1", Environment: "test", SecretHash: hex.EncodeToString(sum[:])}}
	principal, err := NewAuthenticator(repository).Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.ProjectID != "prj_1" || !repository.touched {
		t.Fatalf("principal=%#v touched=%v", principal, repository.touched)
	}
}

func TestAuthenticateRejectsWrongSecret(t *testing.T) {
	repository := &apiKeyRepositoryStub{key: APIKey{SecretHash: stringsOf("0", 64)}}
	if _, err := NewAuthenticator(repository).Authenticate(context.Background(), "if_live_invalid_credential_123456"); !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("error=%v", err)
	}
}

type apiKeyRepositoryStub struct {
	key     APIKey
	touched bool
}

func (repository *apiKeyRepositoryStub) FindActiveByPrefix(context.Context, string) (APIKey, error) {
	return repository.key, nil
}
func (repository *apiKeyRepositoryStub) TouchLastUsed(context.Context, string, time.Time) error {
	repository.touched = true
	return nil
}
func stringsOf(value string, count int) string {
	result := ""
	for range count {
		result += value
	}
	return result
}
