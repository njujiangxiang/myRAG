package parser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
)

// Parser 处理多种格式的文档解析
type Parser struct{}

// NewParser 创建一个新的文档解析器
func NewParser() *Parser {
	return &Parser{}
}

// ParseResult 包含解析后的文档内容和元数据
type ParseResult struct {
	Content string            `json:"content"`
	Pages   []PageContent     `json:"pages,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// PageContent 表示单页内容
type PageContent struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

// Parse 根据 MIME 类型解析文档内容
func (p *Parser) Parse(data []byte, mimeType string) (*ParseResult, error) {
	switch mimeType {
	case "application/pdf":
		return p.ParsePDF(data)
	case "text/csv":
		return p.ParseCSV(data)
	case "text/markdown", "text/plain":
		return p.ParseText(data)
	case "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return p.ParseWord(data)
	default:
		// 尝试作为文本解析
		return p.ParseText(data)
	}
}

// ParsePDF 使用 ledongthuc/pdf (纯 Go) 解析 PDF 文档
func (p *Parser) ParsePDF(data []byte) (*ParseResult, error) {
	// 从内存创建 PDF reader
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("打开 PDF 失败：%w", err)
	}

	var buf bytes.Buffer
	var pages []PageContent

	// 获取页数
	numPages := reader.NumPage()
	if numPages == 0 {
		return &ParseResult{
			Content: "",
			Pages:   []PageContent{},
			Meta:    map[string]string{"type": "pdf", "pages": "0"},
		}, nil
	}

	// 缓存所有页面的字体
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

	// 从每页提取文本
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		text, err := page.GetPlainText(fonts)
		if err != nil {
			return nil, fmt.Errorf("从第 %d 页提取文本失败：%w", i, err)
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

	// 如果没有提取到页面，返回空结果
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

// ParseCSV 解析 CSV 文档
func (p *Parser) ParseCSV(data []byte) (*ParseResult, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1 // 允许可变字段数

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析 CSV 失败：%w", err)
	}

	if len(records) == 0 {
		return &ParseResult{
			Content: "",
			Meta:    map[string]string{"rows": "0"},
		}, nil
	}

	// 将 CSV 转换为可读格式
	var buf bytes.Buffer
	headers := records[0]

	// 写入表头
	buf.WriteString("列：")
	buf.WriteString(strings.Join(headers, ", "))
	buf.WriteString("\n\n")

	// 将数据行作为结构化文本写入
	for i, record := range records[1:] {
		buf.WriteString(fmt.Sprintf("第 %d 行:\n", i+1))
		for j, value := range record {
			if j < len(headers) {
				buf.WriteString(fmt.Sprintf("  %s: %s\n", headers[j], value))
			} else {
				buf.WriteString(fmt.Sprintf("  字段 %d: %s\n", j, value))
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

// ParseText 解析纯文本和 Markdown 文档
func (p *Parser) ParseText(data []byte) (*ParseResult, error) {
	content := string(data)

	// 统计行数和单词数用于元数据
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

// ParseWord 解析 Word 文档 (.doc, .docx)
func (p *Parser) ParseWord(data []byte) (*ParseResult, error) {
	// 使用 docx 库解析
	doc, err := docx.ReadDocxFromMemory(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("打开 Word 文档失败：%w", err)
	}
	defer doc.Close()

	// 提取文本内容
	content := doc.Editable().GetContent()

	// 清理无效 UTF-8 字符（Qdrant 要求所有字符串必须是有效的 UTF-8）
	content = strings.ToValidUTF8(content, "")

	// 统计元数据
	words := len(strings.Fields(content))
	paragraphs := strings.Count(content, "\n\n") + strings.Count(content, "\n") + 1

	return &ParseResult{
		Content: content,
		Meta: map[string]string{
			"type":       "word",
			"words":      fmt.Sprintf("%d", words),
			"paragraphs": fmt.Sprintf("%d", paragraphs),
		},
	}, nil
}

// ChunkOptions 定义文本分块方式
type ChunkOptions struct {
	MaxSize    int    // 最大块大小（字符数）
	Overlap    int    // 块之间的重叠字符数
	Separator  string // 分隔符（默认："\\n\\n"）
	MinSize    int    // 最小块大小（小于该值的块会被合并）
	KeepTables bool   // 尝试保持表格完整
}

// DefaultChunkOptions 返回默认分块选项
func DefaultChunkOptions() *ChunkOptions {
	return &ChunkOptions{
		MaxSize:   500,    // bge-small 上下文长度为 512 tokens，留有余量
		Overlap:   50,     // 重叠部分也相应减小
		Separator: "\n\n",
		MinSize:   100,
	}
}

// Chunk 将内容分割为重叠的块用于嵌入
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

		// 如果在末尾，直接取剩余部分
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

		// 尝试在自然边界分割
		remaining := content[start:end]
		splitPos := strings.LastIndex(remaining, opts.Separator)

		if splitPos > 0 {
			end = start + splitPos
		} else {
			// 尝试在句子边界分割，但确保不会退步太多
			splitPos = strings.LastIndex(remaining, ". ")
			if splitPos > 0 && splitPos > opts.MaxSize/2 {
				// 只有在后半部分找到句子边界才使用
				end = start + splitPos + 1
			}
			// 否则保持 end = start + opts.MaxSize
		}

		chunkContent := strings.TrimSpace(content[start:end])
		if len(chunkContent) >= opts.MinSize {
			// 清理无效 UTF-8 字符
			chunkContent = strings.ToValidUTF8(chunkContent, "")
			chunks = append(chunks, Chunk{
				Content: chunkContent,
				Index:   index,
			})
			index++
		}

		// 移动起始位置，带重叠
		newStart := end - opts.Overlap
		// 确保至少前进 MaxSize - Overlap 的距离
		minAdvance := opts.MaxSize - opts.Overlap
		if newStart < start+minAdvance {
			newStart = start + minAdvance
		}
		start = newStart

		// 防止超出范围
		if start >= len(content) {
			break
		}
	}

	return chunks
}

// Chunk 表示用于嵌入的文本块
type Chunk struct {
	Content string `json:"content"`
	Index   int    `json:"index"`
}
