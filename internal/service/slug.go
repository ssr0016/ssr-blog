package service

import (
	"strconv"
	"strings"
	"unicode"
)

const (
	// MaxSlugLen is the slug column limit.
	MaxSlugLen = 200
	// MaxSlugBaseLen leaves room for a "-N" suffix (up to "-9999999999") under MaxSlugLen.
	MaxSlugBaseLen = 190
)

// latinFold maps lowercase Latin letters with diacritics (Latin-1 Supplement and
// Latin Extended-A) to their ASCII form. Stdlib only: golang.org/x/text would need an ADR.
// Letters not listed here (and every non-Latin script) are treated as separators.
var latinFold = func() map[rune]string {
	groups := []struct{ ascii, letters string }{
		{"a", "àáâãäåāăą"},
		{"c", "çćĉċč"},
		{"d", "ďđð"},
		{"e", "èéêëēĕėęě"},
		{"g", "ĝğġģ"},
		{"h", "ĥħ"},
		{"i", "ìíîïĩīĭįı"},
		{"j", "ĵ"},
		{"k", "ķ"},
		{"l", "ĺļľŀł"},
		{"n", "ñńņňŉ"},
		{"o", "òóôõöøōŏő"},
		{"r", "ŕŗř"},
		{"s", "śŝşš"},
		{"t", "ţťŧ"},
		{"u", "ùúûüũūŭůűų"},
		{"w", "ŵ"},
		{"y", "ýÿŷ"},
		{"z", "źżž"},
		{"ss", "ß"},
		{"ae", "æ"},
		{"oe", "œ"},
		{"th", "þ"},
	}
	m := make(map[rune]string)
	for _, g := range groups {
		for _, r := range g.letters {
			m[r] = g.ascii
		}
	}
	return m
}()

// Slugify turns a title into a URL-safe slug of [a-z0-9-]: lowercase, accents folded,
// every other run of characters collapsed to a single hyphen, no leading or trailing
// hyphen, at most MaxSlugBaseLen characters. It returns "" when the title contains no
// usable letters or digits (for example "!!!" or a non-Latin title); callers must reject that.
func Slugify(title string) string {
	var b strings.Builder
	pendingHyphen := false

	for _, r := range strings.ToLower(title) {
		var out string
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = string(r)
		case unicode.Is(unicode.Mn, r):
			// Combining mark from decomposed input ("e" + U+0301): drop it without
			// splitting the word.
			continue
		default:
			out = latinFold[r]
		}

		if out == "" {
			pendingHyphen = b.Len() > 0
			continue
		}
		if pendingHyphen {
			b.WriteByte('-')
			pendingHyphen = false
		}
		b.WriteString(out)
	}

	slug := b.String()
	if len(slug) > MaxSlugBaseLen { // output is pure ASCII, so byte length == characters
		slug = strings.TrimRight(slug[:MaxSlugBaseLen], "-")
	}
	return slug
}

// WithSuffix returns base for attempt 1 (or less) and "base-N" for attempt N >= 2.
func WithSuffix(base string, n int) string {
	if n <= 1 {
		return base
	}
	return base + "-" + strconv.Itoa(n)
}

// validSlug reports whether s could have been produced by the slug generator: 1..MaxSlugLen
// characters of [a-z0-9-]. Anything else cannot exist, so lookups skip the database.
func validSlug(s string) bool {
	if s == "" || len(s) > MaxSlugLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}
