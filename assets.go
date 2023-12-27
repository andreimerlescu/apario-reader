package main

import (
	`bytes`
	`fmt`
	`net/http`
	`strings`
	`time`

	`github.com/gin-gonic/gin`
)

func getIcon(c *gin.Context) {
	name := c.Param("name")

	filePath := fmt.Sprintf("bundled/assets/icons/%v", name)

	fileData, err := bundled_files.ReadFile(filePath)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	switch {
	case strings.HasSuffix(name, ".js"):
		c.Header("Content-Type", "text/javascript")
	case strings.HasSuffix(name, ".css"):
		c.Header("Content-Type", "text/css")
	case strings.HasSuffix(name, ".woff"):
		c.Header("Content-Type", "font/woff")
	case strings.HasSuffix(name, ".woff2"):
		c.Header("Content-Type", "font/woff2")
	case strings.HasSuffix(name, ".ico"):
		c.Header("Content-Type", "image/x-icon")
	case strings.HasSuffix(name, ".jpg"):
		c.Header("Content-Type", "image/jpeg")
	case strings.HasSuffix(name, ".png"):
		c.Header("Content-Type", "image/png")
	case strings.HasSuffix(name, ".svg"):
		c.Header("Content-Type", "image/svg+xml")
	default:
		c.String(http.StatusInternalServerError, "unsupported image type")
		return
	}

	modTime := time.Now()
	http.ServeContent(c.Writer, c.Request, "", modTime, bytes.NewReader(fileData))
}

func getAsset(c *gin.Context) {
	directory := c.Param("directory")
	filename := c.Param("filename")
	filePath := fmt.Sprintf("bundled/assets/%v/%v", directory, filename)

	fileData, err := bundled_files.ReadFile(filePath)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

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
		c.String(http.StatusInternalServerError, "unsupported image type")
		return
	}

	http.ServeContent(c.Writer, c.Request, "", time.Now(), bytes.NewReader(fileData))
}
