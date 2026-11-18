package customer

import (
	"errors"
	"strings"
	"testing"
)

func TestNewCustomer(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		email     string
		wantName  string
		wantEmail string
		wantErr   error
	}{
		{name: "valid customer", inputName: "Alice", email: "alice@example.com", wantName: "Alice", wantEmail: "alice@example.com"},
		{name: "name with surrounding spaces", inputName: "  Alice  ", email: "alice@example.com", wantName: "Alice", wantEmail: "alice@example.com"},
		{name: "empty name", email: "alice@example.com", wantErr: ErrInvalidName},
		{name: "whitespace-only name", inputName: " \t\n ", email: "alice@example.com", wantErr: ErrInvalidName},
		{name: "name exceeding 150 characters", inputName: strings.Repeat("a", 151), email: "alice@example.com", wantErr: ErrInvalidName},
		{name: "empty email", inputName: "Alice", wantErr: ErrInvalidEmail},
		{name: "email without @", inputName: "Alice", email: "alice.example.com", wantErr: ErrInvalidEmail},
		{name: "email without domain", inputName: "Alice", email: "alice@", wantErr: ErrInvalidEmail},
		{name: "email exceeding 254 bytes", inputName: "Alice", email: "alice@" + strings.Repeat("a", 245) + ".com", wantErr: ErrInvalidEmail},
		{name: "email with surrounding spaces", inputName: "Alice", email: "  alice@example.com  ", wantName: "Alice", wantEmail: "alice@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer, err := NewCustomer(tt.inputName, tt.email)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewCustomer() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if customer != nil {
					t.Errorf("NewCustomer() = %+v, want nil on error", customer)
				}
				return
			}
			if customer == nil {
				t.Fatal("NewCustomer() returned nil customer")
			}
			if customer.Name != tt.wantName {
				t.Errorf("customer.Name = %q, want %q", customer.Name, tt.wantName)
			}
			if customer.Email != tt.wantEmail {
				t.Errorf("customer.Email = %q, want %q", customer.Email, tt.wantEmail)
			}
		})
	}
}
