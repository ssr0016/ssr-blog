package service

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{"plain", "Hello World", "hello-world"},
		{"accents and mixed case", "Café Au Lait", "cafe-au-lait"},
		{"punctuation and repeated spaces", "Hello,   World!!", "hello-world"},
		{"symbols collapse", "C++ & Go", "c-go"},
		{"digits", "2026 Roadmap", "2026-roadmap"},
		{"leading trailing and doubled hyphens", "  --A--B  ", "a-b"},
		{"underscores are separators", "snake_case_title", "snake-case-title"},
		{"upper accents", "ÉCOLE Ñandú", "ecole-nandu"},
		{"sharp s and ligatures", "Straße Æon Œuvre", "strasse-aeon-oeuvre"},
		{"stroke letters", "Łódź Øresund Đà", "lodz-oresund-da"},
		{"latin extended-a", "Škoda Žižek Čapek", "skoda-zizek-capek"},
		{"decomposed accent does not split the word", "Amélie Café", "amelie-cafe"},
		{"apostrophe splits", "Don't Panic", "don-t-panic"},
		{"newlines and tabs", "a\n\tb", "a-b"},
		{"letters mixed with non-latin keep the latin", "Go 言語", "go"},

		// No letters or digits survive: caller must reject.
		{"only punctuation", "!!!", ""},
		{"only spaces", "    ", ""},
		{"empty", "", ""},
		{"japanese", "日本語のタイトル", ""},
		{"arabic", "مرحبا بالعالم", ""},
		{"arabic-indic digits are not ascii digits", "٢٠٢٦", ""},
		{"emoji", "🚀🚀", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.title); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestSlugify_OutputAlphabet(t *testing.T) {
	// Whatever goes in, output is only [a-z0-9-], never starts/ends/doubles a hyphen.
	inputs := []string{
		"Café Au Lait", "<script>alert(1)</script>", "../../etc/passwd", "a/b\\c?d=e&f#g",
		"%20%2F", "ＦＵＬＬＷＩＤＴＨ", "x\u200by", "tab\there", strings.Repeat("é-", 300),
	}
	for _, in := range inputs {
		got := Slugify(in)
		if strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") || strings.Contains(got, "--") {
			t.Errorf("Slugify(%q) = %q has bad hyphens", in, got)
		}
		for _, r := range got {
			isLower := r >= 'a' && r <= 'z'
			isDigit := r >= '0' && r <= '9'
			if !isLower && !isDigit && r != '-' {
				t.Errorf("Slugify(%q) = %q contains %q", in, got, r)
			}
		}
	}
}

func TestSlugify_LongTitleIsTruncated(t *testing.T) {
	got := Slugify(strings.Repeat("a", 300))
	if len(got) != MaxSlugBaseLen {
		t.Errorf("len = %d, want %d", len(got), MaxSlugBaseLen)
	}

	// "word-" repeats every 5 chars, so the 190-char cut lands on a hyphen.
	got = Slugify(strings.Repeat("word ", 60))
	if len(got) > MaxSlugBaseLen {
		t.Errorf("len = %d, want <= %d", len(got), MaxSlugBaseLen)
	}
	if strings.HasSuffix(got, "-") {
		t.Errorf("truncated slug %q ends with a hyphen", got[len(got)-10:])
	}
	if want := MaxSlugBaseLen - 1; len(got) != want {
		t.Errorf("len = %d, want %d (cut at the hyphen, then trimmed)", len(got), want)
	}
}

func TestWithSuffix(t *testing.T) {
	tests := []struct {
		base string
		n    int
		want string
	}{
		{"hello", 1, "hello"},
		{"hello", 2, "hello-2"},
		{"hello", 3, "hello-3"},
		{"hello", 100, "hello-100"},
		{"hello", 0, "hello"},
		{"hello", -5, "hello"},
	}
	for _, tt := range tests {
		if got := WithSuffix(tt.base, tt.n); got != tt.want {
			t.Errorf("WithSuffix(%q, %d) = %q, want %q", tt.base, tt.n, got, tt.want)
		}
	}
}

func TestWithSuffix_FitsColumnAtMaxAttempts(t *testing.T) {
	base := Slugify(strings.Repeat("a", 300))
	for _, n := range []int{2, 9, 10, 99, 100, 999} {
		if got := WithSuffix(base, n); len(got) > MaxSlugLen {
			t.Errorf("WithSuffix(base, %d) has len %d, want <= %d", n, len(got), MaxSlugLen)
		}
	}
}
