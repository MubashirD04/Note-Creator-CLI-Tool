package main

type NoteEntry struct {
	Title             string      `json:"title"`
	CreatedAt         string      `json:"created_at"`
	TranscriptSnippet string      `json:"transcript_snippet"`
	Notes             interface{} `json:"notes"`
}

type NotesFile struct {
	Courses map[string][]NoteEntry `json:"courses"`
}
