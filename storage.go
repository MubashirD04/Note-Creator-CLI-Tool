package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type FileStorage struct {
	FilePath string
}

func NewFileStorage(filePath string) *FileStorage {
	return &FileStorage{FilePath: filePath}
}

func (s *FileStorage) Clear() error {
	emptyJSON := []byte("{\n  \"courses\": {}\n}")
	return os.WriteFile(s.FilePath, emptyJSON, 0644)
}

func (s *FileStorage) UpdateNotes(courseName string, entry NoteEntry) error {
	var fileData NotesFile
	fileData.Courses = make(map[string][]NoteEntry)

	// Try reading existing file
	data, err := os.ReadFile(s.FilePath)
	if err == nil {
		err = json.Unmarshal(data, &fileData)
		if err != nil {
			return fmt.Errorf("failed to parse existing %s: %w", s.FilePath, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if fileData.Courses == nil {
		fileData.Courses = make(map[string][]NoteEntry)
	}

	fileData.Courses[courseName] = append(fileData.Courses[courseName], entry)

	outData, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: write to temp file then rename
	tmpFile := s.FilePath + ".tmp"
	err = os.WriteFile(tmpFile, outData, 0644)
	if err != nil {
		return err
	}

	return os.Rename(tmpFile, s.FilePath)
}
