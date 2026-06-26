package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/yg-codes/proxmox/pkg/onepassword"
)

// allResolvedConfig returns a Config whose Proxmox credentials are non-empty
// plain values, satisfying Validate's required-field checks.
func allPlainConfig() *Config {
	c := &Config{}
	c.Proxmox.Host = "pve.example.com"
	c.Proxmox.Username = "root@pam"
	c.Proxmox.TokenName = "tok"
	c.Proxmox.TokenValue = "secret"
	c.Operations.SnapshotNameMaxLength = 40
	c.Operations.MaxConcurrentSnapshots = 1
	c.Operations.MaxConcurrentVMOps = 1
	return c
}

// TestResolveSecretsSkipsPlainFields verifies that when no PVE_* value is an
// op:// reference, ResolveSecrets leaves everything unchanged and never
// invokes the resolver (no op CLI call).
func TestResolveSecretsSkipsPlainFields(t *testing.T) {
	calls := 0
	restore := onepassword.WithResolver(func(string) (string, error) {
		calls++
		return "should-not-be-used", nil
	})
	defer restore()

	c := allPlainConfig()
	before := *c
	if err := c.ResolveSecrets(); err != nil {
		t.Fatalf("ResolveSecrets with plain values: unexpected error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("resolver invoked %d times, want 0 (no op:// refs present)", calls)
	}
	if c.Proxmox != before.Proxmox {
		t.Fatalf("ResolveSecrets mutated plain config:\n got  %+v\n want %+v", c.Proxmox, before.Proxmox)
	}
}

// TestResolveSecretsResolvesAllRefs verifies each ref field is replaced with
// the resolver's output.
func TestResolveSecretsResolvesAllRefs(t *testing.T) {
	restore := onepassword.WithResolver(func(ref string) (string, error) {
		return "plain:" + ref, nil
	})
	defer restore()

	c := &Config{}
	c.Proxmox.Host = "op://vault/item/host"
	c.Proxmox.Username = "op://vault/item/user"
	c.Proxmox.Password = "op://vault/item/password"
	c.Proxmox.TokenName = "op://vault/item/token_name"
	c.Proxmox.TokenValue = "op://vault/item/token_value"

	if err := c.ResolveSecrets(); err != nil {
		t.Fatalf("ResolveSecrets: unexpected error: %v", err)
	}
	want := map[string]string{
		"Host":       "plain:op://vault/item/host",
		"Username":   "plain:op://vault/item/user",
		"Password":   "plain:op://vault/item/password",
		"TokenName":  "plain:op://vault/item/token_name",
		"TokenValue": "plain:op://vault/item/token_value",
	}
	got := map[string]string{
		"Host": c.Proxmox.Host, "Username": c.Proxmox.Username, "Password": c.Proxmox.Password,
		"TokenName": c.Proxmox.TokenName, "TokenValue": c.Proxmox.TokenValue,
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s = %q, want %q", k, got[k], w)
		}
	}
}

// TestResolveSecretsResolvesMixed verifies plain and ref values coexist: only
// the ref is rewritten, the plain value is untouched.
func TestResolveSecretsMixed(t *testing.T) {
	restore := onepassword.WithResolver(func(string) (string, error) {
		return "resolved-token", nil
	})
	defer restore()

	c := &Config{}
	c.Proxmox.Host = "pve.example.com" // plain
	c.Proxmox.Username = "root@pam"    // plain
	c.Proxmox.TokenName = "tok"        // plain
	c.Proxmox.TokenValue = "op://vault/item/credential" // ref

	if err := c.ResolveSecrets(); err != nil {
		t.Fatalf("ResolveSecrets: unexpected error: %v", err)
	}
	if c.Proxmox.Host != "pve.example.com" {
		t.Errorf("plain Host mutated to %q", c.Proxmox.Host)
	}
	if c.Proxmox.TokenValue != "resolved-token" {
		t.Errorf("ref TokenValue = %q, want %q", c.Proxmox.TokenValue, "resolved-token")
	}
}

// TestResolveSecretsNamesFailingField verifies the error message identifies
// which PVE_* field failed resolution.
func TestResolveSecretsNamesFailingField(t *testing.T) {
	restore := onepassword.WithResolver(func(string) (string, error) {
		return "", errors.New("not signed in")
	})
	defer restore()

	c := &Config{}
	c.Proxmox.Host = "pve.example.com"
	c.Proxmox.Username = "root@pam"
	c.Proxmox.TokenName = "tok"
	c.Proxmox.TokenValue = "op://vault/item/credential" // first ref → fails

	err := c.ResolveSecrets()
	if err == nil {
		t.Fatal("ResolveSecrets with failing resolver: want error, got nil")
	}
	if !strings.Contains(err.Error(), "PVE_TOKEN_VALUE") {
		t.Errorf("error %q does not name the failing field PVE_TOKEN_VALUE", err)
	}
	if !strings.Contains(err.Error(), "not signed in") {
		t.Errorf("error %q does not surface the underlying op error", err)
	}
}
