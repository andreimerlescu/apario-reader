package main

import (
	`encoding/json`
	`errors`
	`fmt`
	`html/template`
	`log`
	`net/http`
	`os`
	`path/filepath`
	`regexp`
	`strings`

	`github.com/gin-gonic/gin`
)

func r_get_page(c *gin.Context) {
	data, err := bundled_files.ReadFile("bundled/assets/html/view-page.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load view-page.html")
		return
	}

	directory := *flag_s_database
	if len(directory) == 0 {
		c.String(http.StatusInternalServerError, "Failed to load view-page.html")
		return
	}

	resolvedPath, symlink_err := resolve_symlink(directory)
	if symlink_err != nil {
		c.String(http.StatusInternalServerError, "Failed to load view-page.html")
		return
	}
	directory = resolvedPath

	log.Printf("using directory %v", directory)

	page_identifier := c.Param("identifier")
	page_identifier = reg_identifier.ReplaceAllString(page_identifier, "") // sanitize input

	template_vars := gin.H{
		"title":           fmt.Sprintf("%v - Page", *flag_s_site_title),
		"company":         *flag_s_site_company,
		"domain":          *flag_s_primary_domain,
		"dark_mode":       gin_is_dark_mode(c),
		"page_identifier": page_identifier,
	}

	mu_page_identifier_document.RLock()
	document_identifier := m_page_identifier_document[page_identifier]
	mu_page_identifier_document.RUnlock()
	template_vars["document_identifier"] = document_identifier

	mu_page_identifier_page_number.RLock()
	page_number := m_page_identifier_page_number[page_identifier]
	mu_page_identifier_page_number.RUnlock()
	template_vars["page_number"] = page_number

	mu_document_total_pages.RLock()
	total_pages := m_document_total_pages[document_identifier]
	mu_document_total_pages.RUnlock()
	template_vars["document_pages"] = total_pages

	mu_document_source_url.RLock()
	source_url := m_document_source_url[document_identifier]
	mu_document_source_url.RUnlock()
	template_vars["url"] = source_url

	mu_document_metadata.RLock()
	metadata := m_document_metadata[document_identifier]
	mu_document_metadata.RUnlock()
	for key, value := range metadata {
		template_vars["meta_"+key] = value
	}

	metadata["Page"] = fmt.Sprintf("Page %d of %d", page_number, total_pages)
	metadata["Source URL"] = source_url
	template_vars["i_page"] = int(page_number)
	template_vars["i_total_pages"] = int(total_pages)
	template_vars["metadata"] = metadata
	template_vars["cover_small"] = fmt.Sprintf("/covers/%v/%v/small.jpg", document_identifier, page_identifier)
	template_vars["cover_medium"] = fmt.Sprintf("/covers/%v/%v/medium.jpg", document_identifier, page_identifier)
	template_vars["cover_large"] = fmt.Sprintf("/covers/%v/%v/large.jpg", document_identifier, page_identifier)
	template_vars["cover_original"] = fmt.Sprintf("/covers/%v/%v/original.jpg", document_identifier, page_identifier)
	template_vars["cover_social"] = fmt.Sprintf("/covers/%v/%v/social.jpg", document_identifier, page_identifier)

	var document_directory_name string
	var has_directory_name bool
	mu_document_identifier_directory.RLock()
	document_directory_name, has_directory_name = m_document_identifier_directory[document_identifier]
	mu_document_identifier_directory.RUnlock()
	if !has_directory_name {
		c.String(http.StatusInternalServerError, "no such data directory")
		return
	}

	ocr_path := filepath.Join(".", directory, document_directory_name, "pages", fmt.Sprintf("ocr.%06d.txt", page_number))
	ocr_bytes, ocr_err := os.ReadFile(ocr_path)
	if ocr_err != nil {
		log.Printf("failed to read the ocr_path %v due to error %v", ocr_path, ocr_err)
		c.String(http.StatusInternalServerError, "error executing template", ocr_err)
		return
	}

	document_pdf_path := filepath.Join(".", directory, document_directory_name, fmt.Sprintf("%v.pdf", template_vars["meta_record_number"]))
	document_pdf_info, document_pdf_info_err := os.Stat(document_pdf_path)
	if errors.Is(document_pdf_info_err, os.ErrNotExist) || errors.Is(document_pdf_info_err, os.ErrPermission) || document_pdf_info_err != nil {
		log.Printf("failed to get the info about the document %v pdf path %v due to err %v", document_identifier, document_pdf_path, document_pdf_info_err)
		c.String(http.StatusInternalServerError, "error executing template")
		return
	}

	page_pdf_path := filepath.Join(".", directory, document_directory_name, "pages", fmt.Sprintf("%v_page_%d.pdf", template_vars["meta_record_number"], page_number))
	page_pdf_info, page_pdf_info_err := os.Stat(page_pdf_path)
	if errors.Is(page_pdf_info_err, os.ErrNotExist) || errors.Is(page_pdf_info_err, os.ErrPermission) || page_pdf_info_err != nil {
		log.Printf("failed to get the info about the page %v pdf path %v due to err %v", page_identifier, page_pdf_path, page_pdf_info_err)
		c.String(http.StatusInternalServerError, "error executing template")
		return
	}

	template_vars["document_pdf_bytes"] = document_pdf_info.Size()
	template_vars["page_pdf_bytes"] = page_pdf_info.Size()

	page_data_path := filepath.Join(".", directory, document_directory_name, "pages", fmt.Sprintf("page.%06d.json", page_number))
	page_data_bytes, page_data_err := os.ReadFile(page_data_path)
	if page_data_err != nil {
		log.Printf("failed to read the pages JSON data due to error %v", page_data_err)
		c.String(http.StatusInternalServerError, "error executing template", page_data_err)
		return
	}

	var page_data PendingPage
	page_err := json.Unmarshal(page_data_bytes, &page_data)
	if page_err != nil {
		log.Printf("failed to unmarshal the page JSON due to error %v", page_err)
	}

	ocr := string(ocr_bytes)
	gematria := NewGemScore(ocr)
	template_vars["full_text"] = ocr

	from := c.DefaultQuery("from", "")
	if len(from) > 0 && from != "stumbleinto" {
		match_replacement := "<span class='badge text-bg-warning'>$0</span>"
		search_analysis := AnalyzeQuery(strings.ReplaceAll(from, "%20", ""))
		for _, inclusive := range search_analysis.Ands {
			re_from := regexp.QuoteMeta(inclusive)
			re, re_err := regexp.Compile(`\b` + re_from + `\b`)
			if re_err == nil {
				ocr = re.ReplaceAllString(ocr, match_replacement)
			}
		}
	}

	template_vars["gematria"] = gematria
	template_vars["full_text"] = template.HTML(ocr)

	tmpl := template.Must(template.New("view-page").Funcs(gin_func_map).Parse(string(data)))

	var htmlBuilder strings.Builder
	if err := tmpl.Execute(&htmlBuilder, template_vars); err != nil {
		c.String(http.StatusInternalServerError, "error executing template", err)
		log.Println(err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, htmlBuilder.String())
}

func r_get_download_page(c *gin.Context) {
	sem_pdf_downloads.Acquire()
	defer sem_pdf_downloads.Release()
	// TODO implement PDF downloads of a filename

	data, err := bundled_files.ReadFile("bundled/assets/html/view-document.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load status.html")
		return
	}

	tmpl := template.Must(template.New("view-document").Funcs(gin_func_map).Parse(string(data)))

	var htmlBuilder strings.Builder
	if err := tmpl.Execute(&htmlBuilder, gin.H{
		"title":             fmt.Sprintf("%v - View Document", *flag_s_site_title),
		"company":           *flag_s_site_company,
		"domain":            *flag_s_primary_domain,
		"active_searches":   human_int(int64(sem_concurrent_searches.Len())),
		"i_active_searches": int64(sem_concurrent_searches.Len()),
		"max_searches":      human_int(int64(*flag_i_concurrent_searches)),
		"i_max_searches":    int64(*flag_i_concurrent_searches),
		"in_waiting_room":   human_int(a_i_waiting_room.Load()),
		"i_in_waiting_room": a_i_waiting_room.Load(),
		"dark_mode":         gin_is_dark_mode(c),
	}); err != nil {
		c.String(http.StatusInternalServerError, "error executing template", err)
		log.Println(err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, htmlBuilder.String())
}
