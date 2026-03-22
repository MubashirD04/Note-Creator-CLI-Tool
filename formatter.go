package main

import (
	"bytes"
	"fmt"
	"strings"
)

type MarkdownFormatter struct{}

func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{}
}

func (f *MarkdownFormatter) FormatNote(entry NoteEntry, courseName string) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s\n\n", entry.Title))
	buf.WriteString(fmt.Sprintf("**Course:** %s | **Date:** %s\n\n", courseName, entry.CreatedAt))
	buf.WriteString("tags: #udemy #notes\n\n")

	// Enhanced Template with Table of Contents
	buf.WriteString("## Table of Contents\n")
	buf.WriteString("- [Summary](#summary)\n")
	buf.WriteString("- [Key Concepts](#key-concepts)\n")
	buf.WriteString("- [Detailed Notes](#detailed-notes)\n")
	buf.WriteString("- [Code Examples](#code-examples)\n")
	buf.WriteString("- [Action Items](#action-items)\n\n")

	buf.WriteString("---\n\n### Transcript Snippet\n\n")
	snippetLines := strings.ReplaceAll(strings.TrimSpace(entry.TranscriptSnippet), "\n", "\n> ")
	buf.WriteString(fmt.Sprintf("> %s\n\n---\n\n", snippetLines))

	notesMap, ok := entry.Notes.(map[string]interface{})
	if !ok {
		return buf.String()
	}

	buf.WriteString("<a name=\"summary\"></a>\n")
	if summary, exists := notesMap["summary"]; exists {
		buf.WriteString(fmt.Sprintf("## Summary\n%v\n\n", summary))
	}

	buf.WriteString("<a name=\"key-concepts\"></a>\n")
	if keyConcepts, exists := notesMap["key_concepts"]; exists {
		if kcList, isList := keyConcepts.([]interface{}); isList && len(kcList) > 0 {
			buf.WriteString("## Key Concepts\n")
			for _, kc := range kcList {
				// Handle both new {term, definition} and legacy string formats
				if kcMap, isMap := kc.(map[string]interface{}); isMap {
					term := kcMap["term"]
					definition := kcMap["definition"]
					buf.WriteString(fmt.Sprintf("- **%v**: %v\n", term, definition))
				} else {
					buf.WriteString(fmt.Sprintf("- %v\n", kc))
				}
			}
			buf.WriteString("\n")
		}
	}

	buf.WriteString("<a name=\"detailed-notes\"></a>\n")
	if detailedNotes, exists := notesMap["detailed_notes"]; exists {
		buf.WriteString(fmt.Sprintf("## Detailed Notes\n%v\n\n", detailedNotes))
	}

	buf.WriteString("<a name=\"code-examples\"></a>\n")
	if codeExamples, exists := notesMap["code_examples"]; exists {
		if ceList, isList := codeExamples.([]interface{}); isList && len(ceList) > 0 {
			buf.WriteString("## Code Examples\n")
			for _, ce := range ceList {
				buf.WriteString(fmt.Sprintf("```\n%v\n```\n\n", ce))
			}
		}
	}

	buf.WriteString("<a name=\"action-items\"></a>\n")
	if actionItems, exists := notesMap["action_items"]; exists {
		if aiList, isList := actionItems.([]interface{}); isList && len(aiList) > 0 {
			buf.WriteString("## Action Items\n")
			for _, ai := range aiList {
				buf.WriteString(fmt.Sprintf("- [ ] %v\n", ai))
			}
			buf.WriteString("\n")
		}
	}

	return buf.String()
}
