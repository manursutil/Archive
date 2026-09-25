package main

import (
	"errors"
	"regexp"
)

type Item struct {
	URL     string
	Content string
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

	pathRegex := `\.(txt|pdf|doc|docx|odf|md)$`
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

	switch tp {
	case "url":
		// 1. get url contents
		HTML, err := fetchURL(arg)
		if err != nil {
			return Item{}, err
		}

		content, err := extractText(HTML)
		if err != nil {
			return Item{}, err
		}

		// 2. turn into item object
		item, err := createItem(arg, content)
		return item, err

	case "path":
		// 1. get contents
		// 2. turn into item object
		return Item{}, errors.New("path support not implemented")
	default:
		// Should never reach this
		return Item{}, errors.New("invalid argument")
	}

}

func fetchURL(url string) ([]byte, error) {
	return []byte{}, nil
}

func extractText([]byte) (string, error) {
	return "", nil
}

func createItem(url, content string) (Item, error) {
	if url == "" || content == "" {
		return Item{}, errors.New("cannot create empty item")
	}

	return Item{
		URL:     url,
		Content: content,
	}, nil
}
