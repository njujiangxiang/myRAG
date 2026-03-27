package parser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Parser handles document parsing for various formats
type Parser struct{}

// NewParser creates a new document parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseResult contains the parsed document content and metadata
type ParseResult struct {
	Content string            `json:"content"`
	Pages   []PageContent     `json:"pages,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// PageContent represents content from a single page
type PageContent struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

// Parse parses document content based on MIME type
func (p *Parser) Parse(data []byte, mimeType string) (*ParseResult, error) {
	switch mimeType {
	case "application/pdf":
		return p.ParsePDF(data)
	case "text/csv":
		return p.ParseCSV(data)
	case "text/markdown", "text/plain":
		return p.ParseText(data)
	default:
		// Try to parse as text
		return p.ParseText(data)
	}
}

// ParsePDF parses PDF documents using ledongthuc/pdf (pure Go)
func (p *Parser) ParsePDF(data []byte) (*ParseResult, error) {
	// Create PDF reader from memory
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}

	var buf bytes.Buffer
	var pages []PageContent

	// Get number of pages
	numPages := reader.NumPage()
	if numPages == 0 {
		return &ParseResult{
			Content: "",
			Pages:   []PageContent{},
			Meta:    map[string]string{"type": "pdf", "pages": "0"},
		}, nil
	}

	// Cache fonts for all pages
	fonts := make(map[string]*pdf.Font)
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		for _, name := range page.Fonts() {
			if _, ok := fonts[name]; !ok {
				f := page.Font(name)
				fonts[name] = &f
			}
		}
	}

	// Extract text from each page
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		text, err := page.GetPlainText(fonts)
		if err != nil {
			return nil, fmt.Errorf("failed to extract text from page %d: %w", i, err)
		}

		cleanText := strings.TrimSpace(text)
		if cleanText != "" {
			pages = append(pages, PageContent{
				Number: i,
				Text:   cleanText,
			})
			buf.WriteString(cleanText)
			buf.WriteString("\n\n")
		}
	}

	// If no pages were extracted, return empty result
	if len(pages) == 0 {
		return &ParseResult{
			Content: "",
			Pages:   []PageContent{},
			Meta:    map[string]string{"type": "pdf", "pages": "0"},
		}, nil
	}

	return &ParseResult{
		Content: buf.String(),
		Pages:   pages,
		Meta:    map[string]string{"type": "pdf", "pages": fmt.Sprintf("%d", len(pages))},
	}, nil
}

// ParseCSV parses CSV documents
func (p *Parser) ParseCSV(data []byte) (*ParseResult, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1 // Allow variable field count

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return &ParseResult{
			Content: "",
			Meta:    map[string]string{"rows": "0"},
		}, nil
	}

	// Convert CSV to a readable format
	var buf bytes.Buffer
	headers := records[0]

	// Write headers
	buf.WriteString("Columns: ")
	buf.WriteString(strings.Join(headers, ", "))
	buf.WriteString("\n\n")

	// Write data rows as structured text
	for i, record := range records[1:] {
		buf.WriteString(fmt.Sprintf("Row %d:\n", i+1))
		for j, value := range record {
			if j < len(headers) {
				buf.WriteString(fmt.Sprintf("  %s: %s\n", headers[j], value))
			} else {
				buf.WriteString(fmt.Sprintf("  Field%d: %s\n", j, value))
			}
		}
		buf.WriteString("\n")
	}

	return &ParseResult{
		Content: buf.String(),
		Meta: map[string]string{
			"rows":    fmt.Sprintf("%d", len(records)-1),
			"columns": fmt.Sprintf("%d", len(headers)),
		},
	}, nil
}

// ParseText parses plain text and markdown documents
func (p *Parser) ParseText(data []byte) (*ParseResult, error) {
	content := string(data)

	// Count lines and words for metadata
	lines := strings.Count(content, "\n") + 1
	words := len(strings.Fields(content))

	return &ParseResult{
		Content: content,
		Meta: map[string]string{
			"lines": fmt.Sprintf("%d", lines),
			"words": fmt.Sprintf("%d", words),
		},
	}, nil
}

// ChunkOptions defines how text should be chunked
type ChunkOptions struct {
	MaxSize    int    // Maximum chunk size in characters
	Overlap    int    // Overlap between chunks in characters
	Separator  string // Separator for splitting (default: "\n\n")
	MinSize    int    // Minimum chunk size (smaller chunks are merged)
	KeepTables bool   // Try to keep tables together
}

// DefaultChunkOptions returns default chunk options
func DefaultChunkOptions() *ChunkOptions {
	return &ChunkOptions{
		MaxSize:   4000,
		Overlap:   200,
		Separator: "\n\n",
		MinSize:   100,
	}
}

// Chunk splits content into overlapping chunks for embedding
func (p *Parser) Chunk(content string, opts *ChunkOptions) []Chunk {
	if opts == nil {
		opts = DefaultChunkOptions()
	}

	if len(content) <= opts.MaxSize {
		return []Chunk{{
			Content: content,
			Index:   0,
		}}
	}

	var chunks []Chunk
	start := 0
	index := 0

	for start < len(content) {
		end := start + opts.MaxSize

		// If we're at the end, just take the rest
		if end >= len(content) {
			chunkContent := strings.TrimSpace(content[start:])
			if len(chunkContent) >= opts.MinSize {
				chunks = append(chunks, Chunk{
					Content: chunkContent,
					Index:   index,
				})
			}
			break
		}

		// Try to split at a natural boundary
		remaining := content[start:end]
		splitPos := strings.LastIndex(remaining, opts.Separator)

		if splitPos > 0 {
			end = start + splitPos
		} else {
			// Try splitting at sentence boundary
			splitPos = strings.LastIndex(remaining, ". ")
			if splitPos > 0 {
				end = start + splitPos + 1
			}
		}

		chunkContent := strings.TrimSpace(content[start:end])
		if len(chunkContent) >= opts.MinSize {
			chunks = append(chunks, Chunk{
				Content: chunkContent,
				Index:   index,
			})
			index++
		}

		// Move start position with overlap
		start = end - opts.Overlap
		if start < 0 {
			start = 0
		}

		// Prevent infinite loop
		if start >= len(content) {
			break
		}
	}

	return chunks
}

// Chunk represents a text chunk for embedding
type Chunk struct {
	Content string `json:"content"`
	Index   int    `json:"index"`
}
