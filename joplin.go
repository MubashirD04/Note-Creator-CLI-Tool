package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type JoplinClient struct {
	Token string
	URL   string
}

func NewJoplinClient() *JoplinClient {
	token := os.Getenv("JOPLIN_TOKEN")
	if token == "" {
		return nil
	}
	port := os.Getenv("JOPLIN_PORT")
	if port == "" {
		port = "41184"
	}
	return &JoplinClient{
		Token: token,
		URL:   fmt.Sprintf("http://127.0.0.1:%s", port),
	}
}

type JoplinFolder struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type FoldersResponse struct {
	Items   []JoplinFolder `json:"items"`
	HasMore bool           `json:"has_more"`
}

func (c *JoplinClient) GetOrCreateFolder(title string) (string, error) {
	page := 1
	for {
		url := fmt.Sprintf("%s/folders?token=%s&page=%d", c.URL, c.Token, page)
		resp, err := http.Get(url)
		if err != nil {
			return "", fmt.Errorf("failed fetching folders: %w", err)
		}

		var fResp FoldersResponse
		err = json.NewDecoder(resp.Body).Decode(&fResp)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("failed decoding folders: %w", err)
		}

		for _, item := range fResp.Items {
			if item.Title == title {
				return item.ID, nil
			}
		}

		if !fResp.HasMore {
			break
		}
		page++
	}

	reqBody := map[string]string{"title": title}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/folders?token=%s", c.URL, c.Token)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed creating folder: %w", err)
	}
	defer resp.Body.Close()

	var created JoplinFolder
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("failed decoding new folder: %w", err)
	}

	return created.ID, nil
}

func (c *JoplinClient) CreateNote(title, body, parentId string) error {
	reqBody := map[string]string{
		"title":     title,
		"body":      body,
		"parent_id": parentId,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/notes?token=%s", c.URL, c.Token)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed creating note: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(out))
	}
	return nil
}
