package editorjs

import (
	`fmt`
	`log`
	`strconv`
	`strings`
)

func Markdown(input string, options ...Options) string {
	var markdownOptions Options

	if len(options) > 0 {
		markdownOptions = options[0]
	}

	var result []string
	editorJSAST := ParseEditorJSON(input)

	for _, el := range editorJSAST.Blocks {

		data := el.Data

		switch el.Type {

		case "header":
			result = append(result, markdown_gen_header(data))

		case "paragraph":
			result = append(result, data.Text)

		case "list":
			result = append(result, markdown_gen_list(data))

		case "image":
			result = append(result, markdown_gen_image(data, markdownOptions))

		case "rawTool":
			result = append(result, data.HTML)

		case "delimiter":
			result = append(result, "---")

		case "table":
			result = append(result, markdown_gen_table(data))

		case "caption":
			result = append(result, markdown_gen_caption(data))

		default:
			log.Println("Unknown data type: " + el.Type)
		}

	}

	return strings.Join(result[:], "\n\n")
}

func markdown_gen_header(el Data) string {
	var result []string

	for i := 0; i < el.Level; i++ {
		result = append(result, "#")
	}

	result = append(result, " "+el.Text)

	return strings.Join(result[:], "")
}

func markdown_gen_list(el Data) string {
	var result []string

	if el.Style == "unordered" {
		for _, el := range el.Items {
			result = append(result, "- "+el)
		}
	} else {
		for i, el := range el.Items {
			n := strconv.Itoa(i+1) + "."
			result = append(result, fmt.Sprintf("%s %s", n, el))
		}
	}

	return strings.Join(result[:], "\n")
}

func markdown_gen_image(el Data, options Options) string {
	classes := options.Image.Classes
	withBorder := classes.WithBorder
	stretched := classes.Stretched
	withBackground := classes.WithBackground

	if withBorder == "" &&
		stretched == "" &&
		withBackground == "" {

		return fmt.Sprintf("![%s](%s)", options.Image.Caption, el.File.URL)
	}

	if withBorder == "" && el.WithBorder {
		withBorder = "editorjs-with-border"
	}

	if stretched == "" && el.Stretched {
		stretched = "editorjs-stretched"
	}

	if withBackground == "" && el.WithBackground {
		withBackground = "editorjs-withBackground"
	}

	return fmt.Sprintf(`<img src="%s" alt="%s" class="%s %s %s" />`, el.File.URL, options.Image.Caption, withBorder, stretched, withBackground)
}

func markdown_gen_table(el Data) string {
	var result []string

	for _, cell := range el.Content {
		row := strings.Join(cell, " | ")
		result = append(result, fmt.Sprintf("| %s |", row))
	}

	return strings.Join(result, "\n")
}

func markdown_gen_caption(el Data) string {
	return fmt.Sprintf("> %s", el.Text)
}
