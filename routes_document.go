package main

import (
	`bytes`
	`fmt`
	`html/template`
	`log`
	`net/http`
	`os`
	`path/filepath`
	`sort`
	`strconv`
	`strings`
	`sync/atomic`
	`time`

	`github.com/gin-gonic/gin`
)

func r_get_documents(c *gin.Context) {
	s_page := c.DefaultQuery("page", "1")
	s_limit := c.DefaultQuery("limit", "12")

	var page int
	var limit int

	i_page, i_page_err := strconv.Atoi(s_page)
	if i_page_err != nil {
		page = 1
	} else {
		page = i_page
	}

	i_limit, i_limit_err := strconv.Atoi(s_limit)
	if i_limit_err != nil {
		limit = 12
	} else {
		limit = i_limit
	}

	total_documents := len(m_index_document_identifier)
	total_pages := (total_documents + limit - 1) / limit // Calculate total pages, rounding up

	start_index := (page - 1) * limit
	end_index := start_index + limit

	var page_identifiers []string
	var increased_index_by atomic.Int64
	for i := start_index; i < end_index; i++ {
		if len(m_index_document_identifier[int64(i)]) > 0 {
			document_identifier := m_index_document_identifier[int64(i)]
			mu_document_identifier_cover_page_identifier.RLock()
			cover_page_identifier := m_document_identifier_cover_page_identifier[document_identifier]
			mu_document_identifier_cover_page_identifier.RUnlock()
			page_identifiers = append(page_identifiers, cover_page_identifier)
		} else {
			total_increases := increased_index_by.Add(1)
			if total_increases < 3 {
				end_index++
			}
		}
	}

	data, err := bundled_files.ReadFile("bundled/assets/html/all-documents.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load all-documents.html")
		return
	}

	tmpl := template.Must(template.New("all-documents").Funcs(gin_func_map).Parse(string(data)))

	var htmlBuilder strings.Builder
	if err := tmpl.Execute(&htmlBuilder, gin.H{
		"title":                    fmt.Sprintf("%v - All Documents", *flag_s_site_title),
		"company":                  *flag_s_site_company,
		"domain":                   *flag_s_primary_domain,
		"page":                     human_int(int64(page)),
		"i_page":                   page,
		"limit":                    human_int(int64(limit)),
		"i_limit":                  limit,
		"total_documents":          human_int(int64(total_documents)),
		"i_total_documents":        total_documents,
		"total_pages":              human_int(int64(total_pages)),
		"i_total_pages":            total_pages,
		"page_identifiers":         page_identifiers,
		"dark_mode":                gin_is_dark_mode(c),
		"from":                     c.DefaultQuery("from", ""),
		"page_loading_svg_img_src": template.HTML(svg_page_loading_img_src),
	}); err != nil {
		c.String(http.StatusInternalServerError, "error executing template", err)
		log.Println(err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, htmlBuilder.String())
}

func r_get_view_document(c *gin.Context) {
	data, err := bundled_files.ReadFile("bundled/assets/html/view-document.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load view-document.html")
		return
	}

	document_identifier := c.Param("identifier")
	document_identifier = reg_identifier.ReplaceAllString(document_identifier, "") // sanitize input

	tmpl := template.Must(template.New("view-document").Funcs(gin_func_map).Parse(string(data)))

	var template_vars gin.H
	template_vars = gin.H{
		"title":     fmt.Sprintf("%v - View Document", *flag_s_site_title), // TODO add record_number and title from metadata here
		"company":   *flag_s_site_company,
		"domain":    *flag_s_primary_domain,
		"from":      c.DefaultQuery("from", ""),
		"dark_mode": gin_is_dark_mode(c),
	}

	template_vars["page_loading_svg_img_src"] = template.HTML(svg_page_loading_img_src)

	mu_document_metadata.RLock()
	metadata := m_document_metadata[document_identifier]
	mu_document_metadata.RUnlock()
	for key, value := range metadata {
		template_vars["meta_"+key] = value
	}

	mu_document_total_pages.RLock()
	total_pages := m_document_total_pages[document_identifier]
	mu_document_total_pages.RUnlock()
	template_vars["document_pages"] = total_pages

	var pages map[string]uint // map[PageIdentifier]PageNumber
	var has_pages bool
	mu_document_page_identifiers_pgno.RLock()
	pages, has_pages = m_document_page_identifiers_pgno[document_identifier]
	mu_document_page_identifiers_pgno.RUnlock()
	if !has_pages {
		referrer := c.Request.Header.Get("Referer")
		if referrer != "" {
			c.Redirect(http.StatusFound, referrer)
		} else {
			c.Redirect(http.StatusFound, "/")
		}
		return
	}

	type page_data struct {
		PageIdentifier string
		PageNumber     uint
	}
	var s_page_data []page_data
	for page_identifier, page_number := range pages {
		s_page_data = append(s_page_data, page_data{page_identifier, page_number})
	}

	sort.Slice(s_page_data, func(left, right int) bool {
		return s_page_data[left].PageNumber < s_page_data[right].PageNumber
	})

	template_vars["pages"] = s_page_data
	template_vars["page_loading_svg_img_src"] = template.HTML(svg_page_loading_img_src)

	var htmlBuilder strings.Builder
	if err := tmpl.Execute(&htmlBuilder, template_vars); err != nil {
		c.String(http.StatusInternalServerError, "error executing template", err)
		log.Println(err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, htmlBuilder.String())
}

func r_get_download_document(c *gin.Context) {
	sem_pdf_downloads.Acquire()
	defer sem_pdf_downloads.Release()
	// TODO implement PDF downloads of a filename

	directory := *flag_s_database
	if len(directory) == 0 {
		c.String(http.StatusNotFound, fmt.Sprintf("failed to load database %v", directory))
		return
	}

	resolvedPath, symlink_err := resolve_symlink(directory)
	if symlink_err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to load %v", directory))
		return
	}
	directory = resolvedPath
	directory = strings.ReplaceAll(directory, filepath.Join(*flag_s_database, *flag_s_database), *flag_s_database)
	log.Printf("using directory %v", directory)

	document_identifier := c.Param("document_identifier")
	document_identifier = reg_identifier.ReplaceAllString(document_identifier, "") // sanitize input

	filename := c.Param("filename")
	if !reg_pdf_name.MatchString(filename) { // sanitize input
		c.String(http.StatusForbidden, "invalid pdf %v", filename)
		return
	}

	mu_document_identifier_directory.RLock()
	document_directory, is_found := m_document_identifier_directory[document_identifier] // validate document_identifier
	mu_document_identifier_directory.RUnlock()
	if !is_found {
		c.String(http.StatusInternalServerError, "failed to find %v directory checksum", document_identifier)
		return
	}

	pdf_path := filepath.Join(directory, document_directory, filename)
	pdf_path = strings.ReplaceAll(pdf_path, filepath.Join(*flag_s_database, *flag_s_database), *flag_s_database)

	image_info, stat_err := os.Stat(pdf_path)
	if stat_err != nil {
		log.Printf("failed to stat %v due to %v", pdf_path, stat_err)
		c.String(http.StatusNotFound, "no such pdf")
		return
	}

	if image_info.Size() == 0 {
		log.Printf("failed to pass the .Size() > 0 check on %v due", pdf_path)
		c.String(http.StatusInternalServerError, "failed to load pdf")
		return
	}

	file_bytes, file_err := os.ReadFile(pdf_path)
	if file_err != nil {
		c.String(http.StatusInternalServerError, "failed to open %v due to %v", filename, file_err)
		return
	}

	c.Header("Content-Type", "application/pdf")

	modTime := time.Now()
	http.ServeContent(c.Writer, c.Request, filename, modTime, bytes.NewReader(file_bytes))
}
