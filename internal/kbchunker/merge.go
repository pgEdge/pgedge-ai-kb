/*-------------------------------------------------------------------------
 *
 * pgEdge AI Knowledgebase Builder
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package kbchunker

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Regular expressions for splitting HTML tables into rows
var (
	htmlRowOpenRegex  = regexp.MustCompile(`(?i)<tr(\s|>)`)
	htmlRowCloseRegex = regexp.MustCompile(`(?i)</tr\s*>`)
	htmlTheadEndRegex = regexp.MustCompile(`(?i)</thead\s*>`)
)

// RawChunk is an intermediate chunk before final processing.
// It holds the text content and metadata about the structural elements
// contained within the chunk.
type RawChunk struct {
	Text         string
	ElementTypes []string
}

// ChunkConfig holds configuration parameters for chunking.
// The MaxSize and MaxChars values are hard constraints for Ollama compatibility.
type ChunkConfig struct {
	TargetSize     int  // Target words per chunk (default: 250)
	MaxSize        int  // Maximum words per chunk (default: 300) - HARD LIMIT
	MinSize        int  // Minimum words before merging (default: 100)
	MaxChars       int  // Character limit (default: 3000) - HARD LIMIT
	OverlapWords   int  // Overlap between chunks (default: 50)
	PreserveCode   bool // Keep code blocks intact when possible (default: true)
	PreserveTables bool // Keep tables intact when possible (default: true)
}

// DefaultChunkConfig returns the default chunking configuration.
// MaxSize and MaxChars are set to maintain Ollama embedding model compatibility.
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		TargetSize:     TargetChunkSize,
		MaxSize:        MaxChunkSize,
		MinSize:        100, // Minimum before considering merge
		MaxChars:       MaxChunkChars,
		OverlapWords:   OverlapSize,
		PreserveCode:   true,
		PreserveTables: true,
	}
}

// mergeUndersizedChunks combines small chunks with their neighbors.
// This is pass 2 of the hybrid chunking algorithm.
//
// Rules:
// - Chunks below minSize words are candidates for merging
// - Merged chunks must not exceed maxSize words
// - Prefer merging with the next chunk for better reading flow
// - Character limits are also respected
func mergeUndersizedChunks(chunks []RawChunk, minSize, maxSize, maxChars int) []RawChunk {
	if len(chunks) <= 1 {
		return chunks
	}

	var merged []RawChunk
	i := 0

	for i < len(chunks) {
		current := chunks[i]
		currentWords := wordCount(current.Text)

		// Check if current chunk is undersized and can merge with next
		if currentWords < minSize && i+1 < len(chunks) {
			next := chunks[i+1]
			nextWords := wordCount(next.Text)
			combinedText := current.Text + "\n\n" + next.Text
			combinedChars := len(combinedText)

			// Merge if combined size is within limits
			if currentWords+nextWords <= maxSize && combinedChars <= maxChars {
				merged = append(merged, RawChunk{
					Text:         combinedText,
					ElementTypes: append(current.ElementTypes, next.ElementTypes...),
				})
				i += 2 // Skip both chunks
				continue
			}
		}

		// Can't merge, keep as is
		merged = append(merged, current)
		i++
	}

	// Second pass: try to merge trailing undersized chunks backwards
	if len(merged) > 1 {
		merged = mergeTrailingUndersized(merged, minSize, maxSize, maxChars)
	}

	return merged
}

// mergeTrailingUndersized handles the case where the last chunk is undersized.
// It tries to merge it with the preceding chunk.
func mergeTrailingUndersized(chunks []RawChunk, minSize, maxSize, maxChars int) []RawChunk {
	if len(chunks) < 2 {
		return chunks
	}

	lastIdx := len(chunks) - 1
	last := chunks[lastIdx]
	lastWords := wordCount(last.Text)

	// If last chunk is undersized, try to merge with previous
	if lastWords < minSize {
		prev := chunks[lastIdx-1]
		prevWords := wordCount(prev.Text)
		combinedText := prev.Text + "\n\n" + last.Text
		combinedChars := len(combinedText)

		if prevWords+lastWords <= maxSize && combinedChars <= maxChars {
			// Merge with previous
			chunks[lastIdx-1] = RawChunk{
				Text:         combinedText,
				ElementTypes: append(prev.ElementTypes, last.ElementTypes...),
			}
			chunks = chunks[:lastIdx] // Remove last chunk
		}
	}

	return chunks
}

// splitAtSemanticBoundaries splits content into chunks while respecting
// structural element boundaries. This is pass 1 of the hybrid chunking algorithm.
//
// Rules:
// - Never split within a structural element (code block, table, list, blockquote)
// - Prefer splitting at paragraph boundaries
// - Split oversized paragraphs at sentence boundaries
// - Respect maxSize and maxChars limits
func splitAtSemanticBoundaries(elements []StructuralElement, cfg ChunkConfig) []RawChunk {
	var chunks []RawChunk
	var currentChunk RawChunk

	for _, elem := range elements {
		elemWords := wordCount(elem.Content)
		elemChars := len(elem.Content)

		// If element alone exceeds limits, it must be split
		if elemWords > cfg.MaxSize || elemChars > cfg.MaxChars {
			// Flush current chunk first
			if currentChunk.Text != "" {
				chunks = append(chunks, currentChunk)
				currentChunk = RawChunk{}
			}

			// Split the oversized element
			subChunks := splitOversizedElement(elem, cfg)
			chunks = append(chunks, subChunks...)
			continue
		}

		// Check if adding this element would exceed limits
		currentWords := wordCount(currentChunk.Text)
		currentChars := len(currentChunk.Text)

		wouldExceedWords := currentWords+elemWords > cfg.TargetSize
		wouldExceedChars := currentChars+elemChars+2 > cfg.MaxChars // +2 for "\n\n"

		if wouldExceedWords || wouldExceedChars {
			// Flush current chunk
			if currentChunk.Text != "" {
				chunks = append(chunks, currentChunk)
				currentChunk = RawChunk{}
			}
		}

		// Add element to current chunk
		if currentChunk.Text != "" {
			currentChunk.Text += "\n\n" + elem.Content
		} else {
			currentChunk.Text = elem.Content
		}
		currentChunk.ElementTypes = append(currentChunk.ElementTypes, elem.Type.String())
	}

	// Flush final chunk
	if currentChunk.Text != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// splitOversizedElement handles elements that exceed the size limits.
// Different strategies are used based on element type.
func splitOversizedElement(elem StructuralElement, cfg ChunkConfig) []RawChunk {
	switch elem.Type {
	case Paragraph:
		return splitParagraphAtSentences(elem.Content, cfg)
	case CodeBlock:
		return splitCodeBlockAtLines(elem.Content, cfg)
	case Table:
		if htmlTableOpenRegex.MatchString(elem.Content) {
			return splitHTMLTableAtRows(elem.Content, cfg)
		}
		return splitTableAtRows(elem.Content, cfg)
	case List:
		return splitListAtItems(elem.Content, cfg)
	case Blockquote:
		return splitBlockquoteAtLines(elem.Content, cfg)
	default:
		// Fallback: split at word boundaries
		return splitAtWordBoundaries(elem.Content, elem.Type.String(), cfg)
	}
}

// splitParagraphAtSentences splits a paragraph at sentence boundaries.
func splitParagraphAtSentences(content string, cfg ChunkConfig) []RawChunk {
	var chunks []RawChunk
	var currentText strings.Builder

	sentences := splitIntoSentences(content)

	for _, sentence := range sentences {
		sentenceWords := wordCount(sentence)
		currentWords := wordCount(currentText.String())
		currentChars := currentText.Len()

		// Check if adding this sentence would exceed limits
		wouldExceedWords := currentWords+sentenceWords > cfg.MaxSize
		wouldExceedChars := currentChars+len(sentence) >= cfg.MaxChars

		if wouldExceedWords || wouldExceedChars {
			if currentText.Len() > 0 {
				chunks = append(chunks, RawChunk{
					Text:         strings.TrimSpace(currentText.String()),
					ElementTypes: []string{"paragraph"},
				})
				currentText.Reset()
			}
		}

		if currentText.Len() > 0 {
			currentText.WriteString(" ")
		}
		currentText.WriteString(sentence)
	}

	// Flush remaining content
	if currentText.Len() > 0 {
		chunks = append(chunks, RawChunk{
			Text:         strings.TrimSpace(currentText.String()),
			ElementTypes: []string{"paragraph"},
		})
	}

	return chunks
}

// splitIntoSentences splits text into sentences based on punctuation.
func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)

		// Check for sentence-ending punctuation
		if r == '.' || r == '!' || r == '?' {
			// Look ahead to see if this is really the end of a sentence
			if i+1 < len(text) {
				next := text[i+1]
				// If followed by space and uppercase, or end of text, it's a sentence end
				if next == ' ' || next == '\n' {
					sentences = append(sentences, strings.TrimSpace(current.String()))
					current.Reset()
				}
			} else {
				// End of text
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}
	}

	// Flush remaining content
	if current.Len() > 0 {
		remaining := strings.TrimSpace(current.String())
		if remaining != "" {
			sentences = append(sentences, remaining)
		}
	}

	return sentences
}

// splitCodeBlockAtLines splits a code block at line boundaries.
func splitCodeBlockAtLines(content string, cfg ChunkConfig) []RawChunk {
	lines := strings.Split(content, "\n")
	var chunks []RawChunk
	var currentLines []string

	// Check if this is a fenced code block
	isFenced := len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```")
	openingFence := ""
	if isFenced && len(lines) > 0 {
		openingFence = lines[0]
		lines = lines[1:] // Remove opening fence from processing
		// Also remove closing fence if present
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
	}

	for _, line := range lines {
		currentLines = append(currentLines, line)
		currentText := strings.Join(currentLines, "\n")
		currentWords := wordCount(currentText)
		currentChars := len(currentText)

		// Check if we've exceeded limits
		if currentWords >= cfg.MaxSize || currentChars >= cfg.MaxChars {
			// Create chunk (re-add fences for fenced code blocks)
			chunkText := currentText
			if isFenced {
				chunkText = openingFence + "\n" + currentText + "\n```"
			}
			chunks = append(chunks, RawChunk{
				Text:         chunkText,
				ElementTypes: []string{"code_block"},
			})
			currentLines = nil
		}
	}

	// Flush remaining lines
	if len(currentLines) > 0 {
		chunkText := strings.Join(currentLines, "\n")
		if isFenced {
			chunkText = openingFence + "\n" + chunkText + "\n```"
		}
		chunks = append(chunks, RawChunk{
			Text:         chunkText,
			ElementTypes: []string{"code_block"},
		})
	}

	return chunks
}

// splitTableAtRows splits a table at row boundaries.
func splitTableAtRows(content string, cfg ChunkConfig) []RawChunk {
	lines := strings.Split(content, "\n")
	var chunks []RawChunk
	var header []string
	var currentRows []string

	// First two lines are typically header and separator
	if len(lines) >= 2 {
		header = lines[:2]
		lines = lines[2:]
	}

	for _, line := range lines {
		currentRows = append(currentRows, line)
		currentText := strings.Join(append(header, currentRows...), "\n")
		currentWords := wordCount(currentText)
		currentChars := len(currentText)

		// Check if we've exceeded limits
		if currentWords >= cfg.MaxSize || currentChars >= cfg.MaxChars {
			chunks = append(chunks, RawChunk{
				Text:         currentText,
				ElementTypes: []string{"table"},
			})
			currentRows = nil
		}
	}

	// Flush remaining rows
	if len(currentRows) > 0 {
		chunkText := strings.Join(append(header, currentRows...), "\n")
		chunks = append(chunks, RawChunk{
			Text:         chunkText,
			ElementTypes: []string{"table"},
		})
	}

	// If no chunks were created (table smaller than limits), return the original
	if len(chunks) == 0 {
		chunks = append(chunks, RawChunk{
			Text:         content,
			ElementTypes: []string{"table"},
		})
	}

	return chunks
}

// splitHTMLTableAtRows splits an HTML table at <tr> boundaries. The
// markup before the first body row (the <table> tag and any <thead>) is
// repeated at the start of every chunk and the markup after the last row
// is repeated at the end, so each chunk stays a self-describing table.
func splitHTMLTableAtRows(content string, cfg ChunkConfig) []RawChunk {
	bodyStart := 0
	if loc := htmlTheadEndRegex.FindStringIndex(content); loc != nil {
		bodyStart = loc[1]
	}
	firstRow := htmlRowOpenRegex.FindStringIndex(content[bodyStart:])
	rowEnds := htmlRowCloseRegex.FindAllStringIndex(content, -1)
	if firstRow == nil || len(rowEnds) == 0 {
		return splitAtWordBoundaries(content, "table", cfg)
	}
	bodyStart += firstRow[0]
	bodyEnd := rowEnds[len(rowEnds)-1][1]
	if bodyEnd <= bodyStart {
		return splitAtWordBoundaries(content, "table", cfg)
	}

	header := content[:bodyStart]
	footer := content[bodyEnd:]
	// Drop the repeated wrapper if it would leave no room for rows
	if len(header)+len(footer) > cfg.MaxChars/2 ||
		wordCount(header)+wordCount(footer) > cfg.MaxSize/2 {
		header, footer = "", ""
	}

	// Each row runs from one <tr> to the next
	body := content[bodyStart:bodyEnd]
	starts := htmlRowOpenRegex.FindAllStringIndex(body, -1)
	rows := make([]string, 0, len(starts))
	for k, loc := range starts {
		end := len(body)
		if k+1 < len(starts) {
			end = starts[k+1][0]
		}
		if row := strings.TrimSpace(body[loc[0]:end]); row != "" {
			rows = append(rows, row)
		}
	}

	var chunks []RawChunk
	var currentRows []string
	build := func(rs []string) string {
		return strings.TrimSpace(header + strings.Join(rs, "\n") + footer)
	}
	for _, row := range rows {
		candidate := build(append(currentRows, row))
		fits := len(candidate) <= cfg.MaxChars &&
			wordCount(candidate) <= cfg.MaxSize
		if !fits && len(currentRows) > 0 {
			chunks = append(chunks, RawChunk{
				Text:         build(currentRows),
				ElementTypes: []string{"table"},
			})
			currentRows = nil
		}
		currentRows = append(currentRows, row)
	}
	if len(currentRows) > 0 {
		chunks = append(chunks, RawChunk{
			Text:         build(currentRows),
			ElementTypes: []string{"table"},
		})
	}

	return chunks
}

// splitListAtItems splits a list at item boundaries.
func splitListAtItems(content string, cfg ChunkConfig) []RawChunk {
	lines := strings.Split(content, "\n")
	var chunks []RawChunk
	var currentItem strings.Builder
	var currentItems []string

	for _, line := range lines {
		// Check if this is a new list item (top-level)
		if listItemRegex.MatchString(line) && getIndentation(line) == 0 {
			// Save previous item if any
			if currentItem.Len() > 0 {
				currentItems = append(currentItems, currentItem.String())
				currentItem.Reset()
			}

			// Check if current items exceed limits
			currentText := strings.Join(currentItems, "\n")
			if wordCount(currentText) >= cfg.MaxSize || len(currentText) >= cfg.MaxChars {
				chunks = append(chunks, RawChunk{
					Text:         currentText,
					ElementTypes: []string{"list"},
				})
				currentItems = nil
			}
		}

		if currentItem.Len() > 0 {
			currentItem.WriteString("\n")
		}
		currentItem.WriteString(line)
	}

	// Add final item
	if currentItem.Len() > 0 {
		currentItems = append(currentItems, currentItem.String())
	}

	// Flush remaining items
	if len(currentItems) > 0 {
		chunks = append(chunks, RawChunk{
			Text:         strings.Join(currentItems, "\n"),
			ElementTypes: []string{"list"},
		})
	}

	return chunks
}

// splitBlockquoteAtLines splits a blockquote at line boundaries.
func splitBlockquoteAtLines(content string, cfg ChunkConfig) []RawChunk {
	lines := strings.Split(content, "\n")
	var chunks []RawChunk
	var currentLines []string

	for _, line := range lines {
		currentLines = append(currentLines, line)
		currentText := strings.Join(currentLines, "\n")
		currentWords := wordCount(currentText)
		currentChars := len(currentText)

		// Check if we've exceeded limits
		if currentWords >= cfg.MaxSize || currentChars >= cfg.MaxChars {
			chunks = append(chunks, RawChunk{
				Text:         currentText,
				ElementTypes: []string{"blockquote"},
			})
			currentLines = nil
		}
	}

	// Flush remaining lines
	if len(currentLines) > 0 {
		chunks = append(chunks, RawChunk{
			Text:         strings.Join(currentLines, "\n"),
			ElementTypes: []string{"blockquote"},
		})
	}

	return chunks
}

// splitAtWordBoundaries is a fallback for splitting content at word boundaries.
func splitAtWordBoundaries(content, elementType string, cfg ChunkConfig) []RawChunk {
	words := strings.Fields(content)
	var chunks []RawChunk
	var currentWords []string

	for _, word := range words {
		currentWords = append(currentWords, word)
		currentText := strings.Join(currentWords, " ")

		// Check if we've exceeded limits
		if len(currentWords) >= cfg.MaxSize || len(currentText) >= cfg.MaxChars {
			chunks = append(chunks, RawChunk{
				Text:         currentText,
				ElementTypes: []string{elementType},
			})
			currentWords = nil
		}
	}

	// Flush remaining words
	if len(currentWords) > 0 {
		chunks = append(chunks, RawChunk{
			Text:         strings.Join(currentWords, " "),
			ElementTypes: []string{elementType},
		})
	}

	return chunks
}

// enforceHardLimits re-splits any chunk that still exceeds MaxChars or
// MaxSize after the semantic and merge passes. The type-specific splitters
// cannot divide content with no natural boundary (a paragraph with no
// sentence punctuation, a single very long line), and embedding providers
// reject inputs over their token limit, so this is the final guarantee.
func enforceHardLimits(chunks []RawChunk, cfg ChunkConfig) []RawChunk {
	var out []RawChunk
	for _, ch := range chunks {
		if withinLimits(ch.Text, cfg) {
			out = append(out, ch)
			continue
		}
		for _, text := range splitToLimits(ch.Text, cfg) {
			out = append(out, RawChunk{
				Text:         text,
				ElementTypes: ch.ElementTypes,
			})
		}
	}
	return out
}

// withinLimits reports whether text satisfies both hard size limits.
func withinLimits(text string, cfg ChunkConfig) bool {
	return len(text) <= cfg.MaxChars && wordCount(text) <= cfg.MaxSize
}

// splitToLimits packs text into pieces that satisfy the hard limits,
// breaking at line boundaries where possible, then at word boundaries,
// and finally inside a word for tokens longer than MaxChars.
func splitToLimits(text string, cfg ChunkConfig) []string {
	var pieces []string
	var current strings.Builder
	currentWords := 0

	flush := func() {
		if s := strings.TrimSpace(current.String()); s != "" {
			pieces = append(pieces, s)
		}
		current.Reset()
		currentWords = 0
	}
	add := func(part, sep string) {
		partWords := wordCount(part)
		if current.Len() > 0 &&
			(current.Len()+len(sep)+len(part) > cfg.MaxChars ||
				currentWords+partWords > cfg.MaxSize) {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString(sep)
		}
		current.WriteString(part)
		currentWords += partWords
	}

	for _, line := range strings.Split(text, "\n") {
		if withinLimits(line, cfg) {
			add(line, "\n")
			continue
		}
		// The line alone is too big: fall back to words
		flush()
		for _, word := range strings.Fields(line) {
			for _, part := range splitLongWord(word, cfg.MaxChars) {
				add(part, " ")
			}
		}
		flush()
	}
	flush()

	return pieces
}

// splitLongWord cuts a word longer than maxChars bytes into pieces of at
// most maxChars bytes without breaking a UTF-8 sequence.
func splitLongWord(word string, maxChars int) []string {
	if len(word) <= maxChars {
		return []string{word}
	}
	var parts []string
	for len(word) > maxChars {
		cut := maxChars
		for cut > 0 && !utf8.RuneStart(word[cut]) {
			cut--
		}
		if cut == 0 {
			cut = maxChars
		}
		parts = append(parts, word[:cut])
		word = word[cut:]
	}
	if word != "" {
		parts = append(parts, word)
	}
	return parts
}

// wordCount returns the number of words in text.
func wordCount(text string) int {
	return len(strings.Fields(text))
}
