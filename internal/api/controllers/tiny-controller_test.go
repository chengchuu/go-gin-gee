package controllers

import (
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTinyErrorPage(t *testing.T) {
	cases := []struct {
		message   string
		title     string
		content   string
		classname string
	}{
		{"404 Link Not Found", "404 Link Not Found", "This short link could not be found.", "warn"},
		{"404 Link Not Available", "404 Link Not Available", "This short link could not be opened.", "error"},
		{"404 Link Expired", "404 Link Expired", "This one-time link has already been used.", "warn"},
		{"", "404 Link Not Found", "This short link could not be found.", "warn"},
		{"unexpected <error>", "unexpected <error>", "This short link could not be opened. Check the link or contact the person who shared it.", "error"},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			app := gin.New()
			app.LoadHTMLFiles("../../../assets/data/index.tmpl")
			app.GET("/", func(c *gin.Context) { renderTinyError(c, errors.New(tc.message)) })
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			if w.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", w.Code)
			}
			for _, expected := range []string{
				`<h1 id="title">` + html.EscapeString(tc.title) + `</h1>`,
				`<p id="desc">` + tc.content + `</p>`,
				`aria-describedby="desc"`,
				`<main class="base base-` + tc.classname + `"`,
			} {
				if !strings.Contains(w.Body.String(), expected) {
					t.Errorf("response missing %q", expected)
				}
			}
		})
	}
}

func TestTinyTemplateClassname(t *testing.T) {
	for _, tc := range []struct {
		name string
		data gin.H
		want string
	}{
		{"absent", gin.H{}, "info"},
		{"empty", gin.H{"classname": ""}, "info"},
		{"warning", gin.H{"classname": "warn"}, "warn"},
		{"error", gin.H{"classname": "error"}, "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := gin.New()
			app.LoadHTMLFiles("../../../assets/data/index.tmpl")
			app.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.tmpl", tc.data) })
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			if !strings.Contains(w.Body.String(), `<main class="base base-`+tc.want+`"`) {
				t.Fatalf("expected exactly one state class: base-%s", tc.want)
			}
		})
	}
}

func TestTinyTemplateOptionalContent(t *testing.T) {
	for _, content := range []string{"", "<script>alert(1)</script>"} {
		t.Run(content, func(t *testing.T) {
			app := gin.New()
			app.LoadHTMLFiles("../../../assets/data/index.tmpl")
			app.GET("/", func(c *gin.Context) {
				data := gin.H{"title": "Welcome"}
				if content != "" {
					data["content"] = content
				}
				c.HTML(http.StatusOK, "index.tmpl", data)
			})
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			body := w.Body.String()
			if content == "" {
				if strings.Contains(body, `id="desc"`) || strings.Contains(body, "aria-describedby") {
					t.Fatal("description or accessibility reference rendered without content")
				}
			} else if !strings.Contains(body, `<p id="desc">`+html.EscapeString(content)+`</p>`) || !strings.Contains(body, `aria-describedby="desc"`) {
				t.Fatal("content was not escaped or accessibility reference is missing")
			}
		})
	}
}
