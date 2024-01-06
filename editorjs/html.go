package editorjs

import (
	`fmt`
	`log`
	`strconv`
	`strings`
)

func HTML(input string, options ...Options) string {
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
			result = append(result, html_gen_header(data))

		case "paragraph":
			result = append(result, html_gen_paragraph(el.Data))

		case "list":
			result = append(result, html_gen_list(data))

		case "image":
			result = append(result, html_gen_image(data, markdownOptions))

		case "rawTool":
			result = append(result, data.HTML)

		case "delimiter":
			result = append(result, "---")

		case "table":
			result = append(result, markdown_gen_table(data)) // TODO: implement html_gen_table

		case "caption":
			result = append(result, markdown_gen_caption(data)) // TODO: implement html_gen_caption

		default:
			log.Println("Unknown data type: " + el.Type)
		}

	}

	return strings.Join(result[:], "\n\n")
}

func html_gen_header(el Data) string {
	level := strconv.Itoa(el.Level)
	return fmt.Sprintf("<h%s>%s</h%s>", level, el.Text, level)
}

func html_gen_paragraph(el Data) string {
	return fmt.Sprintf("<p>%s</p>", el.Text)
}

func html_gen_list(el Data) string {
	var result []string

	if el.Style == "unordered" {
		result = append(result, "<ul>")

		for _, el := range el.Items {
			result = append(result, "  <li>"+el+"</li>")
		}

		result = append(result, "</ul>")
	} else {
		result = append(result, "<ol>")

		for _, el := range el.Items {
			result = append(result, "  <li>"+el+"</li>")
		}

		result = append(result, "</ol>")
	}

	return strings.Join(result[:], "\n")
}

func html_gen_image(el Data, options Options) string {
	classes := options.Image.Classes
	withBorder := classes.WithBorder
	stretched := classes.Stretched
	withBackground := classes.WithBackground

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
