package onepassword

import "testing"

func TestIsRef(t *testing.T) {
	cases := map[string]bool{
		"op://SRE/proxmox/token_value": true,
		"op://Vault/Item/field":        true,
		"plain-token-value":            false,
		"":                             false,
		"https://example.com":          false,
	}
	for in, want := range cases {
		if got := IsRef(in); got != want {
			t.Errorf("IsRef(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolveValuePassthrough(t *testing.T) {
	// Non-references must pass through unchanged without invoking the op CLI.
	const plain = "literal-secret"
	got, err := ResolveValue(plain)
	if err != nil {
		t.Fatalf("ResolveValue(%q) unexpected error: %v", plain, err)
	}
	if got != plain {
		t.Errorf("ResolveValue(%q) = %q, want unchanged", plain, got)
	}
}
