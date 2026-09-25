package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type Item struct {
	Source  string
	Content string
}

type ItemRepr struct {
	ID      int
	Source  string
	SavedAt string
}

func parseArg(arg string) (string, error) {
	// determine if argument is a path to a file or a url
	// url: http[s]://whatever.[com | org | co | ai]/something
	// path: path/to/file.[txt | pdf| ...]
	urlRegex := `^https?://`
	foundURL, err := regexp.MatchString(urlRegex, arg)
	if err != nil {
		return "", err
	}

	if foundURL {
		return "url", nil
	}

	pathRegex := `(?i)\.(txt|pdf|doc|docx|odt|odf|md)$`
	foundPath, err := regexp.MatchString(pathRegex, arg)
	if err != nil {
		return "", err
	}

	if foundPath {
		return "path", nil
	}

	return "", errors.New("invalid argument")
}

func getItem(arg string) (Item, error) {
	tp, err := parseArg(arg)
	if err != nil {
		return Item{}, err
	}

	var content string

	switch tp {
	case "url":
		// 1. get url contents
		data, err := fetchURL(arg)
		if err != nil {
			return Item{}, err
		}

		content, err = extractHTML(data)
		if err != nil {
			return Item{}, err
		}

	case "path":
		// 1. get contents
		content, err = extractContentFromFile(arg)
		if err != nil {
			return Item{}, err
		}

	default:
		// Should never reach this
		return Item{}, errors.New("invalid argument")
	}

	// 2. turn into item object
	item, err := createItem(arg, content)
	return item, err
}

func fetchURL(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%d: failed to fetch url (%s)", res.StatusCode, url)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %v", err)
	}

	return data, nil
}

func extractHTML(data []byte) (string, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	var text []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript", "svg":
				return
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

	return strings.Join(text, " "), nil
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

	case ".odt", ".odf":
		return extractODT(path)

	case ".doc":
		return "", errors.New("doc extraction not implemented yet")

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
	// TODO: implement pdf extraction
	return "", errors.New("pdf extraction not implemented yet")
}

func extractDOCX(path string) (string, error) {
	// TODO: implement docx extraction
	return "", errors.New("docx extraction not implemented yet")
}

func extractODT(path string) (string, error) {
	// TODO: implement odt extraction
	return "", errors.New("odt extraction not implemented yet")
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
