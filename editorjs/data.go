package editorjs

type EditorJS struct {
	Blocks []Block `json:"blocks"`
}

type Block struct {
	Type string `json:"type"`
	Data Data   `json:"data"`
}

type Data struct {
	Text           string     `json:"text",omitempty`
	Level          int        `json:"level,omitempty" `
	Style          string     `json:"style,omitempty" `
	Items          []string   `json:"items,omitempty" `
	File           FileData   `json:"file,omitempty" `
	Caption        string     `json:"caption,omitempty"`
	WithBorder     bool       `json:"withBorder,omitempty"`
	Stretched      bool       `json:"stretched,omitempty"`
	WithBackground bool       `json:"withBackground,omitempty"`
	HTML           string     `json:"html,omitempty"`
	Content        [][]string `json:"content,omitempty"`
	Alignment      string     `json:"alignment,omitempty"`
}

type FileData struct {
	URL string `json:"url"`
}

type Options struct {
	Image ImageOptions
}

type ImageOptions struct {
	Classes ImageClasses
	Caption string
}

type ImageClasses struct {
	WithBorder     string
	Stretched      string
	WithBackground string
}
