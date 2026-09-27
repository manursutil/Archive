package main

import (
	"archive/zip"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArg(t *testing.T) {
	for _, tc := range []struct{ arg, want string }{
		{"http://example.com", "url"}, {"https://example.com/file.txt", "url"},
		{"a.txt", "path"}, {"a.MD", "path"}, {"a.pdf", "path"}, {"a.doc", ""},
		{"a.docx", "path"}, {"a.odt", "path"}, {"a.odf", ""},
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
	for _, ext := range []string{"doc", "odf", "csv"} {
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

// writeZip builds a document with one entry, like a minimal docx or odt.
func writeZip(t *testing.T, path, name, content string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDOCX(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.docx")

	writeZip(t, path, "word/document.xml", `<w:document xmlns:w="w"><w:body>
		<w:p><w:r><w:t>Hello</w:t><w:tab/><w:t>world</w:t></w:r></w:p>
		<w:p><w:r><w:t>one</w:t><w:br/><w:t>two &amp; three</w:t></w:r></w:p>
		<w:p><w:r><w:instrText>ignored</w:instrText></w:r></w:p>
	</w:body></w:document>`)

	got, err := extractDOCX(path)
	if want := "Hello\tworld\none\ntwo & three"; err != nil || got != want {
		t.Fatalf("extractDOCX = %q, %v; want %q", got, err, want)
	}

	item, err := getItem(path)
	if err != nil || item.Title != "doc" || item.Content != got {
		t.Fatalf("getItem = %+v, %v", item, err)
	}

	for name, entry := range map[string][2]string{
		"missing entry": {"content.xml", "<x/>"},
		"bad xml":       {"word/document.xml", "<w:t>unclosed"},
	} {
		writeZip(t, path, entry[0], entry[1])
		if _, err := extractDOCX(path); err == nil {
			t.Errorf("%s: no error", name)
		}
	}

	os.WriteFile(path, []byte("not a zip"), 0600)
	if _, err := extractDOCX(path); err == nil {
		t.Error("non-zip accepted")
	}
}

func TestODT(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.odt")

	writeZip(t, path, "content.xml", `<office:document-content xmlns:office="o" xmlns:text="t"><office:body><office:text>
		<text:h>Title</text:h>
		<text:p>a<text:s text:c="3"/>b<text:tab/>c<text:line-break/>d<text:s/>e</text:p>
		<text:p>x &lt; y <text:span>styled</text:span></text:p>
		<text:p>bad count<text:s text:c="99999"/>.</text:p>
	</office:text></office:body></office:document-content>`)

	got, err := extractODT(path)
	if want := "Title\na   b\tc\nd e\nx < y styled\nbad count ."; err != nil || got != want {
		t.Fatalf("extractODT = %q, %v; want %q", got, err, want)
	}

	item, err := getItem(path)
	if err != nil || item.Title != "doc" || item.Content != got {
		t.Fatalf("getItem = %+v, %v", item, err)
	}

	writeZip(t, path, "word/document.xml", "<x/>")
	if _, err := extractODT(path); err == nil {
		t.Error("missing content.xml accepted")
	}
}

// A one-page PDF without an xref table, which poppler rebuilds.
const helloPDF = `%PDF-1.4
1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj
2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj
3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 300 100] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj
4 0 obj << >> stream
BT /F1 12 Tf 10 50 Td (TEXT) Tj ET
endstream endobj
5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj
trailer << /Root 1 0 R >>
%%EOF
`

func TestPDF(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not installed")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "doc.pdf")

	os.WriteFile(path, []byte(strings.Replace(helloPDF, "TEXT", "Hello PDF", 1)), 0600)

	got, err := extractPDF(path)
	if err != nil || got != "Hello PDF" {
		t.Fatalf("extractPDF = %q, %v", got, err)
	}

	item, err := getItem(path)
	if err != nil || item.Title != "doc" || item.Content != got {
		t.Fatalf("getItem = %+v, %v", item, err)
	}

	for name, content := range map[string]string{
		"no text": strings.Replace(helloPDF, "TEXT", "", 1),
		"not pdf": "junk",
	} {
		os.WriteFile(path, []byte(content), 0600)
		if _, err := extractPDF(path); err == nil {
			t.Errorf("%s: no error", name)
		}
	}

	if _, err := extractPDF(filepath.Join(dir, "missing.pdf")); err == nil {
		t.Error("missing file accepted")
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
