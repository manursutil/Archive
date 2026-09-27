package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var client = &http.Client{Timeout: 30 * time.Second}

type Item struct {
	Title   string
	Source  string
	Content string
	HTML    string // raw page only for URLs
}

type ItemRepr struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Source  string `json:"source"`
	SavedAt string `json:"savedAt"`
	Excerpt string `json:"excerpt"`
	HasHTML bool   `json:"hasHtml"`
}

type SearchResult struct {
	ID      int     `json:"id"`
	Title   string  `json:"title"`
	Source  string  `json:"source"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
	HasHTML bool    `json:"hasHtml"`
}

func parseArg(arg string) (string, error) {
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		return "url", nil
	}

	switch strings.ToLower(filepath.Ext(arg)) {
	case ".txt", ".pdf", ".docx", ".odt", ".md":
		return "path", nil
	}

	return "", errors.New("invalid argument")
}

func getItem(arg string) (Item, error) {
	tp, err := parseArg(arg)
	if err != nil {
		return Item{}, err
	}

	var title, content, page string

	switch tp {
	case "url":
		// 1. get url contents
		data, _, err := fetchURL(arg)
		if err != nil {
			return Item{}, err
		}

		title, content, err = extractHTML(data)
		if err != nil {
			return Item{}, err
		}

		base, _ := url.Parse(arg)
		page = inlineAssets(data, base)

	case "path":
		// 1. get contents
		content, err = extractContentFromFile(arg)
		if err != nil {
			return Item{}, err
		}

		title = strings.TrimSuffix(filepath.Base(arg), filepath.Ext(arg))
	}

	// 2. turn into item object
	item, err := createItem(arg, content)
	item.Title = title
	item.HTML = page
	return item, err
}

func fetchURL(url string) ([]byte, string, error) {
	res, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("%d: failed to fetch url (%s)", res.StatusCode, url)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read body: %v", err)
	}

	return data, res.Header.Get("Content-Type"), nil
}

// TODO: maybe move the extract logic to another file?

func extractHTML(data []byte) (string, string, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return "", "", err
	}

	var title string
	var text []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript", "svg":
				return
			case "title":
				if title == "" && n.FirstChild != nil {
					title = strings.TrimSpace(n.FirstChild.Data)
				}
			}
		}

		if n.Type == html.TextNode {
			content := strings.TrimSpace(n.Data)

			if content != "" {
				text = append(text, content)
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(doc)

	return title, strings.Join(text, " "), nil
}

func extractContentFromFile(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".txt":
		return extractTXT(path)

	case ".md":
		return extractMarkdown(path)

	case ".pdf":
		return extractPDF(path)

	case ".docx":
		return extractDOCX(path)

	case ".odt":
		return extractODT(path)

	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

func extractTXT(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func extractMarkdown(path string) (string, error) {
	// For now, markdown is treated as plain text
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func extractPDF(path string) (string, error) {
	out, err := exec.Command("pdftotext", "-enc", "UTF-8", path, "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w (install poppler)", err)
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", errors.New("unable to extarct text from pdf")
	}

	return text, nil
}

func extractDOCX(path string) (string, error) {
	data, err := readZip(path, "word/document.xml")
	if err != nil {
		return "", err
	}

	var words strings.Builder
	intext := false

	dec := xml.NewDecoder(bytes.NewReader(data))

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				intext = true
			case "tab":
				words.WriteByte('\t')
			case "br", "cr":
				words.WriteByte('\n')
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				intext = false
			case "p":
				words.WriteByte('\n')
			}
		case xml.CharData:
			if intext {
				words.Write(t)
			}
		}
	}

	return strings.TrimSpace(words.String()), nil
}

func extractODT(path string) (string, error) {
	data, err := readZip(path, "content.xml")
	if err != nil {
		return "", err
	}

	var words strings.Builder
	depth := 0

	dec := xml.NewDecoder(bytes.NewReader(data))

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p", "h":
				depth++
			case "tab":
				words.WriteByte('\t')
			case "line-break":
				words.WriteByte('\n')
			case "s":
				n := 1
				for _, a := range t.Attr {
					if a.Name.Local == "c" {
						if c, err := strconv.Atoi(a.Value); err == nil && c > 0 && c < 1000 {
							n = c
						}
					}
				}

				words.WriteString(strings.Repeat(" ", n))
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p", "h":
				depth--
				words.WriteByte('\n')
			}
		case xml.CharData:
			if depth > 0 {
				words.Write(t)
			}
		}
	}

	return strings.TrimSpace(words.String()), nil
}

func createItem(source, content string) (Item, error) {
	if source == "" || content == "" {
		return Item{}, errors.New("cannot create empty item")
	}

	return Item{
		Source:  source,
		Content: content,
	}, nil
}
