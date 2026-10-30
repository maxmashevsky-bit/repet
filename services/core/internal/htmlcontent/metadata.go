package htmlcontent

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
)

const MaxBytes = 512 * 1024

// Tokenization reads metadata and plain text only; uploaded code is never evaluated here.
func Inspect(source string) (domain.HTMLMetadata, string, error) {
	m := domain.HTMLMetadata{Bytes: len(source)}
	if source == "" || len(source) > MaxBytes || !utf8.ValidString(source) || strings.ContainsRune(source, 0) {
		return m, "", apperr.ErrBadRequest
	}
	z := html.NewTokenizer(strings.NewReader(source))
	var text strings.Builder
	var title bool
	var ignored string
	markup := false
	for count := 0; count < 50000; count++ {
		kind := z.Next()
		if kind == html.ErrorToken {
			if z.Err() != io.EOF {
				return m, "", apperr.ErrBadRequest
			}
			if !markup {
				return m, "", apperr.ErrBadRequest
			}
			return m, strings.TrimSpace(text.String()), nil
		}
		t := z.Token()
		if kind == html.StartTagToken || kind == html.SelfClosingTagToken {
			markup = true
			if t.Data == "title" {
				title = true
			}
			if t.Data == "script" {
				m.Scripts++
			}
			if t.Data == "script" || t.Data == "style" {
				ignored = t.Data
			}
			attrs := map[string]string{}
			for _, a := range t.Attr {
				attrs[a.Key] = a.Val
			}
			if t.Data == "html" {
				m.Language = truncate(attrs["lang"], 32)
			}
			if t.Data == "meta" && strings.EqualFold(attrs["name"], "description") {
				m.Description = truncate(attrs["content"], 1000)
			}
			for _, key := range []string{"src", "href", "srcset", "action"} {
				if v := attrs[key]; v != "" && !strings.HasPrefix(v, "data:") && !strings.HasPrefix(v, "#") {
					m.ExternalResources++
				}
			}
		} else if kind == html.EndTagToken {
			if t.Data == "title" {
				title = false
			}
			if t.Data == ignored {
				ignored = ""
			}
		} else if kind == html.TextToken {
			if title {
				m.Title = truncate(m.Title+t.Data, 180)
			}
			if ignored == "" && !title && text.Len() < 240000 {
				text.WriteString(truncate(strings.Join(strings.Fields(t.Data), " "), 4000))
				text.WriteByte('\n')
			}
		}
	}
	return m, "", apperr.ErrBadRequest
}
func truncate(v string, n int) string {
	r := []rune(v)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}
func SHA256(v string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(v))) }
