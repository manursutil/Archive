package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// CSS + img helpers

var cssRef = regexp.MustCompile(`url\(\s*['"]?([^'")\s]*)['"]?\s*\)|@import\s*['"]([^'"]*)['"]`)

// embed images and styleshe page is returned unchanged when it references nothing to inline.
func inlineAssets(page []byte, base *url.URL) string {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil || base == nil {
		return string(page)
	}

	changed := false
	cache := map[string]string{}

	var inlineCSS func(css string, base *url.URL) string

	embed := func(ref string, base *url.URL) string {
		ref = strings.TrimSpace(ref)
		if ref == "" || ref[0] == '#' {
			return ref
		}

		u, err := base.Parse(ref)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return ref
		}

		key := u.String()
		if v, ok := cache[key]; ok {
			return v
		}
		cache[key] = key
		changed = true

		data, ct, err := fetchURL(key)
		if err != nil {
			return key
		}

		if ct == "" {
			ct = http.DetectContentType(data)
		}

		if mt, _, _ := mime.ParseMediaType(ct); mt == "text/css" {
			data = []byte(inlineCSS(string(data), u))
		}

		cache[key] = "data:" + strings.ReplaceAll(ct, " ", "") + ";base64," + base64.StdEncoding.EncodeToString(data)
		return cache[key]
	}

	inlineCSS = func(css string, base *url.URL) string {
		return cssRef.ReplaceAllStringFunc(css, func(m string) string {
			s := cssRef.FindStringSubmatch(m)
			if strings.HasPrefix(m, "@import") {
				return `@import url("` + embed(s[2], base) + `")`
			}
			return `url("` + embed(s[1], base) + `")`
		})
	}

	attr := func(n *html.Node, key string) *html.Attribute {
		for i := range n.Attr {
			if n.Attr[i].Key == key {
				return &n.Attr[i]
			}
		}
		return nil
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if a := attr(n, "style"); a != nil {
				a.Val = inlineCSS(a.Val, base)
			}

			switch n.Data {
			case "base":
				if a := attr(n, "href"); a != nil {
					if u, err := base.Parse(a.Val); err == nil {
						base = u
					}
				}

			case "img":
				if a := attr(n, "src"); a != nil {
					a.Val = embed(a.Val, base)
				}
				fallthrough

			case "source":
				if a := attr(n, "srcset"); a != nil {
					a.Key = "data-srcset"
					changed = true
				}

			case "style":
				if c := n.FirstChild; c != nil && c.Type == html.TextNode {
					c.Data = inlineCSS(c.Data, base)
				}

			case "link":
				rel, href := attr(n, "rel"), attr(n, "href")
				if rel == nil || href == nil || !strings.Contains(strings.ToLower(rel.Val), "stylesheet") {
					break
				}

				u, err := base.Parse(href.Val)
				if err != nil {
					break
				}
				cache[u.String()] = u.String()

				data, _, err := fetchURL(u.String())
				if err != nil {
					href.Val = u.String()
					changed = true
					break
				}

				var keep []html.Attribute
				if m := attr(n, "media"); m != nil {
					keep = append(keep, *m)
				}

				n.Data, n.DataAtom, n.Attr = "style", atom.Style, keep
				n.AppendChild(&html.Node{Type: html.TextNode, Data: inlineCSS(string(data), u)})
				changed = true

				return
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	if !changed {
		return string(page)
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return string(page)
	}

	return buf.String()
}

// zip helpers

const maxXMLsize = 50 << 20

func readZip(path, name string) ([]byte, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	f, err := zr.Open(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxXMLsize+1))
	if err != nil {
		return nil, err
	}

	if len(data) > maxXMLsize {
		return nil, fmt.Errorf("%s: ladger than %d bytes", name, maxXMLsize)
	}

	return data, nil
}
