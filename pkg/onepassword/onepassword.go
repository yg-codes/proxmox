// Package onepassword resolves 1Password secret references (op:// URIs) by
// shelling out to the 1Password CLI. It detects the Windows op.exe binary
// first and falls back to the POSIX op binary, so the same code works on
// Windows, Linux, and macOS.
//
// Only the `op read` subcommand is used: any configuration value that is an
// op:// reference is resolved to its plaintext at startup. This requires the
// 1Password CLI to be installed and signed in (`op signin`).
package onepassword

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// refPrefix is the scheme used by 1Password secret references.
const refPrefix = "op://"

// IsRef reports whether v is a 1Password secret reference (op://...).
func IsRef(v string) bool {
	return strings.HasPrefix(v, refPrefix)
}

// opBinary locates the 1Password CLI, preferring the Windows op.exe binary
// and falling back to op. It returns the resolved path or an error if neither
// is found on PATH.
func opBinary() (string, error) {
	for _, name := range []string{"op.exe", "op"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("1Password CLI not found: install 'op' (or 'op.exe') and ensure it is on PATH (https://developer.1password.com/docs/cli/)")
}

// Resolve reads a single op:// secret reference and returns its plaintext
// value. Stderr from the op CLI is surfaced on error so failures such as a
// missing session ("not signed in") are actionable.
func Resolve(ref string) (string, error) {
	bin, err := opBinary()
	if err != nil {
		return "", err
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, "read", ref)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return "", fmt.Errorf("op read %q failed: %s", ref, detail)
		}
		return "", fmt.Errorf("op read %q failed: %w", ref, err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// ResolveValue resolves v if it is an op:// reference, otherwise returns v
// unchanged. This lets configuration values mix plain secrets and 1Password
// references transparently.
func ResolveValue(v string) (string, error) {
	if !IsRef(v) {
		return v, nil
	}
	return Resolve(v)
}
