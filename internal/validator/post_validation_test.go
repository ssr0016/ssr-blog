package validator_test

import (
	"strings"
	"testing"

	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/validator"
)

type notBlankProbe struct {
	V string `validate:"notblank"`
}

func TestNotBlank(t *testing.T) {
	v := validator.New()
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"empty", "", true},
		{"spaces", "   ", true},
		{"newline and tab", "\n\t", true},
		{"non-breaking space", "  ", true},
		{"ideographic space", "　", true},
		{"surrounded by spaces", " a ", false},
		{"plain", "hello", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(&notBlankProbe{V: tt.in})
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
		})
	}
}

func TestCreateAndUpdatePostRequest_RejectNUL(t *testing.T) {
	v := validator.New()
	for _, field := range []string{"title", "content", "excerpt"} {
		c, u := validCreate(), validUpdate()
		switch field {
		case "title":
			c.Title, u.Title = "a\x00b", "a\x00b"
		case "content":
			c.Content, u.Content = "a\x00b", "a\x00b"
		case "excerpt":
			c.Excerpt, u.Excerpt = "a\x00b", "a\x00b"
		}
		if err := v.Validate(&c); err == nil {
			t.Errorf("create with NUL in %s: want validation error", field)
		}
		if err := v.Validate(&u); err == nil {
			t.Errorf("update with NUL in %s: want validation error", field)
		}
	}
}

func validCreate() model.CreatePostRequest {
	return model.CreatePostRequest{Title: "Hello", Content: "Body"}
}

func validUpdate() model.UpdatePostRequest {
	return model.UpdatePostRequest{Title: "Hello", Content: "Body", Status: model.PostStatusDraft}
}

func TestCreatePostRequest_Validation(t *testing.T) {
	v := validator.New()
	tests := []struct {
		name    string
		mutate  func(*model.CreatePostRequest)
		wantErr bool
	}{
		{"minimal valid", func(*model.CreatePostRequest) {}, false},
		{"status omitted defaults later", func(r *model.CreatePostRequest) { r.Status = "" }, false},
		{"status draft", func(r *model.CreatePostRequest) { r.Status = "draft" }, false},
		{"status published", func(r *model.CreatePostRequest) { r.Status = "published" }, false},
		{"status archived", func(r *model.CreatePostRequest) { r.Status = "archived" }, true},
		{"status wrong case", func(r *model.CreatePostRequest) { r.Status = "Published" }, true},

		{"title missing", func(r *model.CreatePostRequest) { r.Title = "" }, true},
		{"title blank", func(r *model.CreatePostRequest) { r.Title = "   " }, true},
		{"title 200", func(r *model.CreatePostRequest) { r.Title = strings.Repeat("a", 200) }, false},
		{"title 201", func(r *model.CreatePostRequest) { r.Title = strings.Repeat("a", 201) }, true},
		{"title 200 multibyte chars", func(r *model.CreatePostRequest) { r.Title = strings.Repeat("é", 200) }, false},
		{"title 201 multibyte chars", func(r *model.CreatePostRequest) { r.Title = strings.Repeat("é", 201) }, true},

		{"content missing", func(r *model.CreatePostRequest) { r.Content = "" }, true},
		{"content blank", func(r *model.CreatePostRequest) { r.Content = " \n\t " }, true},
		{"content 100000", func(r *model.CreatePostRequest) { r.Content = strings.Repeat("a", 100000) }, false},
		{"content 100001", func(r *model.CreatePostRequest) { r.Content = strings.Repeat("a", 100001) }, true},
		{"content 100000 multibyte chars", func(r *model.CreatePostRequest) { r.Content = strings.Repeat("日", 100000) }, false},
		{"content is stored as written incl. html", func(r *model.CreatePostRequest) { r.Content = "<script>alert(1)</script>" }, false},

		{"excerpt omitted", func(r *model.CreatePostRequest) { r.Excerpt = "" }, false},
		{"excerpt 500", func(r *model.CreatePostRequest) { r.Excerpt = strings.Repeat("a", 500) }, false},
		{"excerpt 501", func(r *model.CreatePostRequest) { r.Excerpt = strings.Repeat("a", 501) }, true},

		{"cover omitted", func(r *model.CreatePostRequest) { r.CoverImageURL = "" }, false},
		{"cover https", func(r *model.CreatePostRequest) { r.CoverImageURL = "https://x.io/a.png" }, false},
		{"cover http", func(r *model.CreatePostRequest) { r.CoverImageURL = "http://x.io/a.png" }, false},
		{"cover javascript scheme", func(r *model.CreatePostRequest) { r.CoverImageURL = "javascript:alert(1)" }, true},
		{"cover ftp scheme", func(r *model.CreatePostRequest) { r.CoverImageURL = "ftp://x.io/a.png" }, true},
		{"cover data uri", func(r *model.CreatePostRequest) { r.CoverImageURL = "data:image/png;base64,AAAA" }, true},
		{"cover not a url", func(r *model.CreatePostRequest) { r.CoverImageURL = "not a url" }, true},
		{"cover scheme without host", func(r *model.CreatePostRequest) { r.CoverImageURL = "https://" }, true},
		{"cover 2048", func(r *model.CreatePostRequest) {
			r.CoverImageURL = "https://x.io/" + strings.Repeat("a", 2048-len("https://x.io/"))
		}, false},
		{"cover 2049", func(r *model.CreatePostRequest) {
			r.CoverImageURL = "https://x.io/" + strings.Repeat("a", 2049-len("https://x.io/"))
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreate()
			tt.mutate(&req)
			err := v.Validate(&req)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdatePostRequest_Validation(t *testing.T) {
	v := validator.New()
	tests := []struct {
		name    string
		mutate  func(*model.UpdatePostRequest)
		wantErr bool
	}{
		{"valid", func(*model.UpdatePostRequest) {}, false},
		{"status published", func(r *model.UpdatePostRequest) { r.Status = "published" }, false},
		{"status required on update", func(r *model.UpdatePostRequest) { r.Status = "" }, true},
		{"status archived", func(r *model.UpdatePostRequest) { r.Status = "archived" }, true},
		{"title blank", func(r *model.UpdatePostRequest) { r.Title = "  " }, true},
		{"title 201", func(r *model.UpdatePostRequest) { r.Title = strings.Repeat("a", 201) }, true},
		{"content blank", func(r *model.UpdatePostRequest) { r.Content = "" }, true},
		{"content 100001", func(r *model.UpdatePostRequest) { r.Content = strings.Repeat("a", 100001) }, true},
		{"excerpt 501", func(r *model.UpdatePostRequest) { r.Excerpt = strings.Repeat("a", 501) }, true},
		{"cover javascript scheme", func(r *model.UpdatePostRequest) { r.CoverImageURL = "javascript:alert(1)" }, true},
		{"cover cleared", func(r *model.UpdatePostRequest) { r.CoverImageURL = "" }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validUpdate()
			tt.mutate(&req)
			err := v.Validate(&req)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
