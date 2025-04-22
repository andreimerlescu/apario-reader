package main

import (
	"fmt"
	"html/template"
	"maps"
	"strings"

	"github.com/gin-gonic/gin"
)

// render_partial_template is a function defined for gin html templates that allows you to render a file from the
// bundled/assets/templates/*.html where * is the first argument of the function call
//
//	{{ render_partial "head" .dark_mode }}
//
// the .dark_mode is a second required parameter for all templates, regardless if the template component actually
// uses a dark mode switcher like
//
//	{{ if eq .dark_mode "1" }}dark{{else}}light{{end}}
//
// this special function will use the gin_func_vars type to store a list of special variables
// that need to be accessible from within the partial template. When writing to the gin_func_vars method, it is
// critical that you use the mu_gin_func_vars to .RLock(), .RUnlock(), .Lock() and .Unlock() the mutex before
// reading and writing to the gin_func_vars map for threadsafety. These variables defined can then be used
// within the template called sample you'd define myvar as
//
//	{{ .myvar }}
//
// when defining .myvar you would do something like:
//
//	mu_gin_func_vars.RLock()
//	existing_vars, have_vars := gin_func_vars['sample']
//	mu_gin_func_vars.RUnlock()
//	if !have_vars || len(existing_vars) == 0 {
//	  mu_gin_func_vars.Lock()
//	  gin_func_map['sample'] = gin.H{
//	    "myvar":     "hello world",
//	  }
//	  mu_gin_func_vars.Unlock()
//	}
//
// the best place to define this would actually be inside the NewWebServer method since it has a sync.Once called
// once_server_start that ensure that you're defining and modifying the gin_func_vars map on every page request.
// For example, from within the NewWebServer method, where new routes are defined:
//
//	r.GET("/new-route", getMyNewRoute)
//
// you would define the template variables used for /new-route below this line (or above it). Consider having one
// section where these are set to prevent excessive lock/unlock on the mutex.
func render_partial_template(path string, dark_mode string) template.HTML {
	body, err := compile_partial_template(path, dark_mode)
	if err != nil {
		return template.HTML("Error: " + err.Error())
	}
	return body
}

func compile_partial_template(path string, dark_mode string) (template.HTML, error) {
	filename := fmt.Sprintf("bundled/assets/templates/%v.html", path)
	data, bundle_err := bundled_files.ReadFile(filename)
	if bundle_err != nil {
		return "", fmt.Errorf("failed to load %v due to err %v", filename, bundle_err)
	}

	mu_gin_func_map.RLock()
	tmpl := template.Must(template.New(path).Funcs(gin_func_map).Parse(string(data)))
	mu_gin_func_map.RUnlock()

	var existing_vars = make(gin.H)
	mu_gin_func_vars.RLock()
	ev, have_vars := gin_func_vars[path]
	mu_gin_func_vars.RUnlock()
	if !have_vars || len(ev) == 0 {
		existing_vars = gin.H{
			"title":        *flag_s_site_title,
			"company":      *flag_s_site_company,
			"domain":       *flag_s_primary_domain,
			"is_dark_mode": dark_mode,
		}
	} else {
		existing_vars = maps.Clone(ev)
	}
	existing_vars["is_dark_mode"] = dark_mode // override with argument value
	existing_vars["loading_svg_img_src"] = template.HTML(svg_page_loading_img_src)
	var htmlBuilder strings.Builder
	template_err := tmpl.Execute(&htmlBuilder, maps.Clone(existing_vars))

	if template_err != nil {
		return "", fmt.Errorf("error executing template: %v", template_err)
	}
	return template.HTML(htmlBuilder.String()), nil
}

// render_document_card is used for all-documents showing the cover page of a document
func render_document_card(cover_page_identifier string, dark_mode string) template.HTML {
	body, err := compile_page_card(cover_page_identifier, dark_mode, "document", "")
	if err != nil {
		return template.HTML("Error: " + err.Error())
	}
	return body
}

// render_page_card is used when viewing a document's pages, this is the per-page card
func render_page_card(page_identifier string, dark_mode string) template.HTML {
	body, err := compile_page_card(page_identifier, dark_mode, "page", "")
	if err != nil {
		return template.HTML("Error: " + err.Error())
	}
	return body
}

// render_page_card is used when viewing a document's pages, this is the per-page card
func render_page_card_from(page_identifier string, dark_mode string, from string) template.HTML {
	body, err := compile_page_card(page_identifier, dark_mode, "page", from)
	if err != nil {
		return template.HTML("Error: " + err.Error())
	}
	return body
}

// compile_page_card builds a template and returns the populated HTML parent_object controls the bundled/assets/component rendered
func compile_page_card(page_identifier string, dark_mode string, parent_object string, from string) (template.HTML, error) {
	filename := fmt.Sprintf("bundled/assets/components/%v-card.html", parent_object)
	data, bundle_err := bundled_files.ReadFile(filename)
	if bundle_err != nil {
		return "", fmt.Errorf("failed to load %v due to err %v", filename, bundle_err)
	}

	path := fmt.Sprintf("page-%v", page_identifier)

	tmpl := template.Must(template.New(path).Funcs(gin_func_map).Parse(string(data)))

	mu_gin_func_vars.RLock()
	existing_vars, have_vars := gin_func_vars[path]
	mu_gin_func_vars.RUnlock()
	if !have_vars || len(existing_vars) == 0 {
		existing_vars = gin.H{
			"title":   *flag_s_site_title,
			"company": *flag_s_site_company,
			"domain":  *flag_s_primary_domain,
		}
	}

	existing_vars["is_dark_mode"] = dark_mode // override with argument value
	existing_vars["page_identifier"] = page_identifier

	existing_vars["from"] = from

	mu_page_identifier_document.RLock()
	document_identifier := m_page_identifier_document[page_identifier]
	mu_page_identifier_document.RUnlock()
	existing_vars["document_identifier"] = document_identifier

	mu_page_identifier_page_number.RLock()
	page_number := m_page_identifier_page_number[page_identifier]
	mu_page_identifier_page_number.RUnlock()
	existing_vars["page_number"] = page_number

	mu_document_total_pages.RLock()
	total_pages := m_document_total_pages[document_identifier]
	mu_document_total_pages.RUnlock()
	existing_vars["document_pages"] = total_pages

	mu_document_source_url.RLock()
	source_url := m_document_source_url[document_identifier]
	mu_document_source_url.RUnlock()
	existing_vars["url"] = source_url

	mu_document_metadata.RLock()
	metadata := m_document_metadata[document_identifier]
	mu_document_metadata.RUnlock()
	for key, value := range metadata {
		existing_vars["meta_"+key] = value
	}

	existing_vars["loading_svg_img_src"] = template.HTML(svg_page_loading_img_src)

	existing_vars["cover_small"] = fmt.Sprintf("/covers/%v/%v/small.jpg", document_identifier, page_identifier)
	existing_vars["cover_medium"] = fmt.Sprintf("/covers/%v/%v/medium.jpg", document_identifier, page_identifier)
	existing_vars["cover_large"] = fmt.Sprintf("/covers/%v/%v/large.jpg", document_identifier, page_identifier)
	existing_vars["cover_original"] = fmt.Sprintf("/covers/%v/%v/original.jpg", document_identifier, page_identifier)
	existing_vars["cover_social"] = fmt.Sprintf("/covers/%v/%v/social.jpg", document_identifier, page_identifier)

	var htmlBuilder strings.Builder

	mu_gin_func_vars.RLock()
	template_err := tmpl.Execute(&htmlBuilder, existing_vars)
	mu_gin_func_vars.RUnlock()

	if template_err != nil {
		return "", fmt.Errorf("error executing template: %v", template_err)
	}
	return template.HTML(htmlBuilder.String()), nil
}

func render_page_detail(identifier string, dark_mode string) template.HTML {
	body, err := compile_page_detail(identifier, dark_mode)
	if err != nil {
		return template.HTML("Error: " + err.Error())
	}
	return body
}

func compile_page_detail(identifier string, dark_mode string) (template.HTML, error) {

	return template.HTML(""), nil
}
