package main

import (
	`bytes`
	`fmt`
	`io`
	`io/fs`
	`log`
	`net/http`
	`os`
	`path/filepath`
	`strings`
	`time`

	`github.com/gin-gonic/gin`
)

func r_get_asset(c *gin.Context) {
	sem_asset_requests.Acquire()
	defer sem_asset_requests.Release()

	defer func(c *gin.Context) {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}(c)

	directory := c.Param("directory")
	filename := c.Param("filename")
	filePath := fmt.Sprintf("bundled/assets/%v/%v", directory, filename)

	file, file_err := bundled_files.Open(filePath)
	defer func(filePath string, file fs.File) { // ensure the file handler is closed
		err := file.Close()
		if err != nil { // handle the close error if applicable
			log.Printf("failed to close r_get_asset file %v due to err %v", filePath, err)
			return
		}
	}(filePath, file) // pass in the filePath and the file into the defer func

	if file_err != nil { // received an err on the file
		log.Printf("received an err on r_get_asset bundled_files.Open(%v) with err %v", filePath, file_err)
		c.String(http.StatusInternalServerError, file_err.Error())
		return
	}

	file_info, info_err := file.Stat() // get info about file
	if info_err != nil {               // error getting info
		log.Printf("received an err on r_get_asset file('%v').Stat() with err %v", filePath, info_err)
		c.String(http.StatusInternalServerError, info_err.Error())
		return
	}

	fileETag := fmt.Sprintf("%x", file_info.ModTime().UnixNano())
	if match := c.GetHeader("If-None-Match"); match == fileETag {
		c.Status(http.StatusNotModified) // dont need to serve the asset, tell the browser to rely on its cache
		return
	}
	c.Header("ETag", fileETag) // set the tag to prevent future load from re-requesting the same asset over again

	// set the return type of data based on the filename suffix
	switch {
	case strings.HasSuffix(filename, ".csv"):
		c.Header("Content-Type", "text/csv")
	case strings.HasSuffix(filename, ".eot"):
		c.Header("Content-Type", "application/vnd.ms-fontobject")
	case strings.HasSuffix(filename, ".epub"):
		c.Header("Content-Type", "application/epub+zip")
	case strings.HasSuffix(filename, ".gif"):
		c.Header("Content-Type", "image/gif")
	case strings.HasSuffix(filename, ".otf"):
		c.Header("Content-Type", "font/otf")
	case strings.HasSuffix(filename, ".pdf"):
		c.Header("Content-Type", "application/pdf")
	case strings.HasSuffix(filename, ".txt"):
		c.Header("Content-Type", "text/plain")
	case strings.HasSuffix(filename, ".js"):
		c.Header("Content-Type", "text/javascript")
	case strings.HasSuffix(filename, ".css"):
		c.Header("Content-Type", "text/css")
	case strings.HasSuffix(filename, ".woff"):
		c.Header("Content-Type", "font/woff")
	case strings.HasSuffix(filename, ".woff2"):
		c.Header("Content-Type", "font/woff2")
	case strings.HasSuffix(filename, ".ico"):
		c.Header("Content-Type", "image/x-icon")
	case strings.HasSuffix(filename, ".jpg"):
		c.Header("Content-Type", "image/jpeg")
	case strings.HasSuffix(filename, ".png"):
		c.Header("Content-Type", "image/png")
	case strings.HasSuffix(filename, ".svg"):
		c.Header("Content-Type", "image/svg+xml")
	case strings.HasSuffix(filename, ".map"):
		c.Header("Content-Type", "application/json")
	default:
		c.String(http.StatusInternalServerError, "unsupported asset type")
		return
	}

	file_bytes, copy_err := io.ReadAll(file) // get all of the bytes inside the file
	if copy_err != nil {
		log.Printf("received an err on r_get_asset io.ReadAll(file('%v')) with err %v", filePath, copy_err)
		c.String(http.StatusInternalServerError, copy_err.Error())
		return
	}

	c.Header("Cache-Control", fmt.Sprintf("max-age=%v", *flag_i_cache_control_assets_seconds)) // dont have the browser re-request this file for seconds

	http.ServeContent(c.Writer, c.Request, "", time.Now(), bytes.NewReader(file_bytes))
}

func r_get_database_page_image(c *gin.Context) {
	sem_image_views.Acquire()
	defer sem_image_views.Release()

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

	page_identifier := c.Param("page_identifier")
	page_identifier = reg_identifier.ReplaceAllString(page_identifier, "") // sanitize input

	size := c.Param("size")
	if !reg_image_size.MatchString(size) { // sanitize input
		c.String(http.StatusForbidden, "invalid size %v", size)
		return
	}

	log.Printf("using document_identifier = %v ; page_identifier = %v ; size = %v", document_identifier, page_identifier, size)

	acceptable_image_size := (strings.HasPrefix(size, "original") ||
		strings.HasPrefix(size, "large") ||
		strings.HasPrefix(size, "medium") ||
		strings.HasPrefix(size, "small") ||
		strings.HasPrefix(size, "social")) && strings.HasSuffix(size, ".jpg")

	if !acceptable_image_size { // validate size
		c.String(http.StatusNotFound, "no such page found")
		return
	}

	document_directory, is_found := m_document_identifier_directory[document_identifier] // validate document_identifier
	if !is_found {
		c.String(http.StatusInternalServerError, "failed to find %v-%v.%v", document_identifier, page_identifier, size)
		return
	}

	mode := gin_is_dark_mode(c)
	if mode == "1" {
		mode = "dark"
	} else {
		mode = "light"
	}

	mu_page_identifier_page_number.RLock()
	page_number, page_number_defined := m_page_identifier_page_number[page_identifier] // validate page_identifier
	mu_page_identifier_page_number.RUnlock()
	if !page_number_defined {
		c.String(http.StatusInternalServerError, "dont know which page number belongs to %v/%v/%v", document_directory, document_identifier, page_identifier)
		return
	}

	image_name := fmt.Sprintf("page.%v.%06d.%v", mode, page_number, size)
	image_path := filepath.Join(directory, document_directory, "pages", image_name) // %v/%v/pages/
	image_path = strings.ReplaceAll(image_path, filepath.Join(*flag_s_database, *flag_s_database), *flag_s_database)

	image_info, stat_err := os.Stat(image_path)
	if stat_err != nil {
		log.Printf("failed to stat %v due to %v", image_path, stat_err)
		c.String(http.StatusNotFound, "no such cover")
		return
	}

	fileETag := fmt.Sprintf("%x", image_info.ModTime().UnixNano())
	if match := c.GetHeader("If-None-Match"); match == fileETag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Header("ETag", fileETag)

	if image_info.Size() == 0 {
		log.Printf("failed to pass the .Size() > 0 check on %v due", image_path)
		c.String(http.StatusInternalServerError, "failed to load cover")
		return
	}

	file_bytes, file_err := os.ReadFile(image_path)
	if file_err != nil {
		c.String(http.StatusInternalServerError, "failed to open %v due to %v", image_name, file_err)
		return
	}

	c.Header("Content-Type", "image/jpg")
	c.Header("Cache-Control", fmt.Sprintf("max-age=%v", *flag_i_cache_control_database_seconds)) // dont have the browser re-request this file for seconds
	http.ServeContent(c.Writer, c.Request, image_name, time.Now(), bytes.NewReader(file_bytes))
}
