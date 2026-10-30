package htmlcontent

import (
	"strings"
	"testing"
)

func TestInspect(t *testing.T) {
	source := `<!doctype html><html lang="ru"><head><title>A &amp; B</title><meta name="description" content="Math"><style>.x{}</style></head><body><h1>Задача 7</h1><p>5 + 2</p><img src="https://example.test/x.png"><script>SECRET_ANSWER=7</script></body></html>`
	m, text, err := Inspect(source)
	if err != nil {
		t.Fatal(err)
	}
	if m.Title != "A & B" || m.Description != "Math" || m.Language != "ru" || m.Scripts != 1 || m.ExternalResources != 1 || m.Bytes != len(source) {
		t.Fatal(m)
	}
	if !strings.Contains(text, "5 + 2") || strings.Contains(text, "SECRET_ANSWER") || strings.Contains(text, ".x{}") {
		t.Fatal(text)
	}
	for _, bad := range []string{"", strings.Repeat("x", MaxBytes+1), "\x00<html>", "plain text", string([]byte{0xff, 0x00})} {
		if _, _, err := Inspect(bad); err == nil {
			t.Fatal("accepted invalid source")
		}
	}
	if SHA256(source) != SHA256(source) || len(SHA256(source)) != 64 {
		t.Fatal("digest")
	}
}
