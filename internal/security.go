package internal

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode"
)

const forbiddenShellChars = ";|&$()`<>\n\r"

func hasForbiddenShellChars(payload string) bool {
	return strings.ContainsAny(payload, forbiddenShellChars)
}

func splitShellArgs(payload string) ([]string, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil, fmt.Errorf("empty command payload")
	}
	if hasForbiddenShellChars(payload) {
		return nil, fmt.Errorf("command contains forbidden shell metacharacters")
	}
	var args []string
	var b strings.Builder
	inQuote := false
	flush := func() {
		if b.Len() > 0 {
			args = append(args, b.String())
			b.Reset()
		}
	}
	for _, r := range payload {
		switch {
		case r == '"':
			inQuote = !inQuote
		case !inQuote && unicode.IsSpace(r):
			flush()
		default:
			b.WriteRune(r)
		}
	}
	if inQuote {
		return nil, fmt.Errorf("unclosed double quote in command")
	}
	flush()
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command payload")
	}
	return args, nil
}

func redactCommandForLog(cmd string) string {
	if cmd == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(cmd))
	preview := cmd
	if len(preview) > 32 {
		preview = preview[:32] + "..."
	}
	return fmt.Sprintf("%q sha256=%x", preview, sum[:8])
}

func summarizeAllowlist(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "(empty-deny-all)"
	}
	if strings.Contains(raw, "*") {
		return "wildcard"
	}
	parts := strings.Split(raw, ",")
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%d entries sha256=%x", len(parts), sum[:8])
}
