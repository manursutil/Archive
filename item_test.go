package main

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArg(t *testing.T) {
	for _, tc := range []struct{ arg, want string }{
		{"http://example.com", "url"}, {"https://example.com/file.txt", "url"},
		{"a.txt", "path"}, {"a.MD", "path"}, {"a.pdf", "path"}, {"a.doc", "path"},
		{"a.docx", "path"}, {"a.odt", "path"}, {"a.odf", "path"},
		{"", ""}, {"a.csv", ""}, {"a.txt.bak", ""},
	} {
		t.Run(tc.arg, func(t *testing.T) {
			got, err := parseArg(tc.arg)

			if got != tc.want || (err != nil) != (tc.want == "") {
				t.Fatalf("parseArg(%q) = %q, %v", tc.arg, got, err)
			}
		})
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()

	for _, ext := range []string{".txt", ".MD"} {
		path := filepath.Join(dir, "note"+ext)
		const content = "# A note\nHello, 世界!\n"

		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}

		got, err := getItem(path)
		if err != nil || got != (Item{Title: "note", Source: path, Content: content}) {
			t.Fatalf("getItem = %+v, %v", got, err)
		}

		if _, err := getItem(filepath.Join(dir, "missing"+ext)); err == nil {
			t.Fatal("missing file accepted")
		}
	}
	for _, ext := range []string{"pdf", "doc", "docx", "odt", "odf", "csv"} {
		if _, err := extractContentFromFile("note." + ext); err == nil {
			t.Fatalf("%s unexpectedly supported", ext)
		}
	}

	if _, err := getItem("invalid"); err == nil {
		t.Fatal("invalid argument accepted")
	}

	for _, item := range []Item{{}, {Source: "source"}, {Content: "content"}} {
		if _, err := createItem(item.Source, item.Content); err == nil {
			t.Fatalf("empty item accepted: %+v", item)
		}
	}
}

// A transport keeps HTTP tests local and can also simulate a broken response body.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type brokenBody struct{}

func (brokenBody) Read([]byte) (int, error) { return 0, errors.New("broken body") }
func (brokenBody) Close() error             { return nil }

func TestURL(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })

	for _, tc := range []struct {
		name, html, title, want, errText string
		status                           int
		broken                           bool
	}{
		{name: "text", status: 200, title: "Greeting", html: `<html><head><title> Greeting </title><style>hidden</style></head><body> <h1>Hello &amp; goodbye</h1><script>hidden</script><noscript>hidden</noscript><svg><text>hidden</text></svg><p>World</p> </body></html>`, want: "Greeting Hello & goodbye World"},
		{name: "malformed HTML", status: 200, html: `<p>Hello <b>world`, want: "Hello world"},
		{name: "empty", status: 200, errText: "cannot create empty item"},
		{name: "status", status: 404, errText: "404: failed to fetch url"},
		{name: "transport", errText: "offline"},
		{name: "body", status: 200, broken: true, errText: "read body: broken body"},
		{name: "deep HTML", status: 200, html: strings.Repeat("<div>", 513), errText: "exceeds 512 nodes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://example.com" {
					t.Fatalf("unexpected URL: %s", r.URL)
				}

				if tc.status == 0 {
					return nil, errors.New("offline")
				}

				var body io.ReadCloser = io.NopCloser(strings.NewReader(tc.html))

				if tc.broken {
					body = brokenBody{}
				}

				return &http.Response{StatusCode: tc.status, Body: body, Header: make(http.Header)}, nil
			})

			got, err := getItem("https://example.com")

			if tc.errText != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errText) {
					t.Fatalf("got %+v, %v; want error %q", got, err, tc.errText)
				}
			} else if err != nil || got != (Item{Title: tc.title, Source: "https://example.com", Content: tc.want, HTML: tc.html}) {
				t.Fatalf("got %+v, %v", got, err)
			}
		})
	}
}

func TestInlineAssets(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })

	assets := map[string][2]string{
		"https://example.com/a/pic.png":  {"image/png", "PNG"},
		"https://example.com/a/site.css": {"text/css", `@import "more.css"; body { background: url(bg.gif) }`},
		"https://example.com/a/more.css": {"text/css", `@import url(site.css); p { color: red }`},
		"https://example.com/a/bg.gif":   {"image/gif", "GIF"},
	}

	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		a, ok := assets[r.URL.String()]
		if !ok {
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(a[1])), Header: http.Header{"Content-Type": {a[0]}}}, nil
	})

	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	base, _ := url.Parse("https://example.com/a/page.html")
	got := inlineAssets([]byte(`<link rel="stylesheet" href="site.css" media="screen"><img src="pic.png" srcset="pic-2x.png 2x"><img src="/gone.png"><div style="background:url('bg.gif')"></div><svg><rect fill="url(#g)"/></svg>`), base)

	for _, want := range []string{
		`<style media="screen">@import url("data:text/css;base64,` + b64(`@import url("https://example.com/a/site.css"); p { color: red }`) + `"); body { background: url("data:image/gif;base64,` + b64("GIF") + `") }</style>`,
		`<img src="data:image/png;base64,` + b64("PNG") + `" data-srcset="pic-2x.png 2x"/>`,
		`<img src="https://example.com/gone.png"/>`,
		`style="background:url(&#34;data:image/gif;base64,` + b64("GIF") + `&#34;)"`,
		`fill="url(#g)"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s\nin %s", want, got)
		}
	}

	if page := "<p>no assets</p>"; inlineAssets([]byte(page), base) != page {
		t.Error("page without assets was rewritten")
	}
}
