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

func NewJoplinClient(token string) *JoplinClient {
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

func (c *JoplinClient) wrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	// Check for connection refused or other common networking errors
	errMsg := err.Error()
	if bytes.Contains([]byte(errMsg), []byte("connection refused")) || 
	   bytes.Contains([]byte(errMsg), []byte("dial tcp")) {
		return fmt.Errorf("\n❌ Joplin connection failed during %s.\n"+
			"Is Joplin running and is the Web Clipper API enabled?\n"+
			"Search settings for 'Web Clipper' to ensure it is active.\n"+
			"Default Port: 41184 (using: %s)\n"+
			"Original Error: %w", op, c.URL, err)
	}
	return fmt.Errorf("%s: %w", op, err)
}

type JoplinFolder struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type FoldersResponse struct {
	Items   []JoplinFolder `json:"items"`
	HasMore bool           `json:"has_more"`
}

func (c *JoplinClient) ListFolders() ([]JoplinFolder, error) {
	var allFolders []JoplinFolder
	page := 1
	for {
		url := fmt.Sprintf("%s/folders?token=%s&page=%d", c.URL, c.Token, page)
		resp, err := http.Get(url)
		if err != nil {
			return nil, c.wrapError("fetching folders", err)
		}

		var fResp FoldersResponse
		err = json.NewDecoder(resp.Body).Decode(&fResp)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed decoding folders: %w", err)
		}

		allFolders = append(allFolders, fResp.Items...)

		if !fResp.HasMore {
			break
		}
		page++
	}
	return allFolders, nil
}

func (c *JoplinClient) GetOrCreateFolder(title string) (string, error) {
	folders, err := c.ListFolders()
	if err != nil {
		return "", err
	}

	for _, item := range folders {
		if item.Title == title {
			return item.ID, nil
		}
	}

	reqBody := map[string]string{"title": title}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/folders?token=%s", c.URL, c.Token)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", c.wrapError("creating folder", err)
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
		return c.wrapError("creating note", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(out))
	}
	return nil
}

type JoplinNote struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	UpdatedTime int64  `json:"updated_time"`
}

type NotesResponse struct {
	Items   []JoplinNote `json:"items"`
	HasMore bool         `json:"has_more"`
}

func (c *JoplinClient) GetFolderNotes(folderId string) ([]JoplinNote, error) {
	var allNotes []JoplinNote
	page := 1
	for {
		url := fmt.Sprintf("%s/folders/%s/notes?token=%s&page=%d&fields=id,title,updated_time", c.URL, folderId, c.Token, page)
		resp, err := http.Get(url)
		if err != nil {
			return nil, c.wrapError("fetching notes", err)
		}

		var nResp NotesResponse
		err = json.NewDecoder(resp.Body).Decode(&nResp)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed decoding notes: %w", err)
		}

		allNotes = append(allNotes, nResp.Items...)

		if !nResp.HasMore {
			break
		}
		page++
	}
	return allNotes, nil
}

type JoplinNoteBody struct {
	Body        string `json:"body"`
	CreatedTime int64  `json:"created_time"`
}

func (c *JoplinClient) GetNoteBody(noteId string) (JoplinNoteBody, error) {
	url := fmt.Sprintf("%s/notes/%s?token=%s&fields=body,created_time", c.URL, noteId, c.Token)
	resp, err := http.Get(url)
	if err != nil {
		return JoplinNoteBody{}, c.wrapError("fetching note body", err)
	}
	defer resp.Body.Close()

	var bodyResp JoplinNoteBody
	err = json.NewDecoder(resp.Body).Decode(&bodyResp)
	if err != nil {
		return JoplinNoteBody{}, fmt.Errorf("failed decoding note body: %w", err)
	}
	return bodyResp, nil
}
