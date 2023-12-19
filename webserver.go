package main

import (
	"bytes"
	"context"
	"crypto/tls"
	`errors`
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
)

func NewWebServer(ctx context.Context, ch_done chan struct{}) {
	once_server_start.Do(func() {
		defer func() {
			ch_done <- struct{}{}
		}()

		// Rate Limiting
		defaultRateLimiter := tollbooth.NewLimiter(*flag_f_rate_limit, &limiter.ExpirableOptions{
			DefaultExpirationTTL: time.Duration(*flag_i_rate_limit_entry_ttl) * time.Second,
			ExpireJobInterval:    time.Duration(*flag_i_rate_limit_cleanup_delay) * time.Second,
		})

		assetRateLimiter := tollbooth.NewLimiter(*flag_f_asset_rate_limit, &limiter.ExpirableOptions{
			DefaultExpirationTTL: time.Duration(*flag_i_asset_rate_limit_entry_ttl) * time.Second,
			ExpireJobInterval:    time.Duration(*flag_i_asset_rate_limit_cleanup_delay) * time.Second,
		})

		// Web Server Configuration
		r := gin.Default()

		// Rate Limit
		r.Use(tollbooth_gin.LimitHandler(defaultRateLimiter))

		// Content Security Policy
		if *flag_b_enable_csp {
			r.Use(middleware_content_security_policy)
		}

		// Cross Origin Resource Sharing (CORS)
		r.Use(func(c *gin.Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusOK)
				return
			}

			c.Next()
		})

		// Respond to a basic ping
		r.Any("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})

		// Content Security Policy
		if *flag_b_enable_csp {
			r.POST(*flag_s_csp_report_uri, func(c *gin.Context) {
				var report map[string]interface{}
				if err := c.ShouldBindJSON(&report); err != nil {
					c.String(http.StatusBadRequest, "Invalid report data")
					return
				}

				log.Println("Received CSP report:", report)

				c.Status(http.StatusOK)
			})
		}

		// Serve all static assets using this entry point
		r.GET("/assets/:name", tollbooth_gin.LimitHandler(assetRateLimiter), getAsset)

		// Go Web Server Index Path
		r.GET("/", func(c *gin.Context) {
			data, err := bundled_files.ReadFile("bundled/assets/html/index.html")
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to load index.html")
				return
			}

			tmpl := template.Must(template.New("index").Parse(string(data)))

			var htmlBuilder strings.Builder
			if err := tmpl.Execute(&htmlBuilder, gin.H{}); err != nil {
				c.String(http.StatusInternalServerError, "error executing template", err)
				log.Println(err)
				return
			}
			c.Header("Content-Type", "text/html; charset=UTF-8")
			c.String(http.StatusOK, htmlBuilder.String())
		})

		// Web App Routes

		// Start HTTP Server
		go func(r *gin.Engine) {
			server := &http.Server{
				Addr:    ":" + strconv.Itoa(*flag_i_webserver_default_port),
				Handler: r,
			}

			go func() {
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					fatalf_log("listen: %s\n", err)
				}
			}()

			<-ctx.Done()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				fatalf_stderr("Server Shutdown Failed:%+v", err)
			}

			log.Println("Server exiting properly")
		}(r)

		// Start HTTPS Server
		go func(r *gin.Engine) {
			cert = loadSSLCertificate()
			startCertReloader()
			server := &http.Server{
				Addr:    ":" + strconv.Itoa(*flag_i_webserver_secure_port),
				Handler: r,
				TLSConfig: &tls.Config{
					GetCertificate: getCertificate,
				},
			}

			go func() {
				if err := server.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
					fatalf_stderr("ListenAndServeTLS(): %s", err)
				}
			}()

			<-ctx.Done()
			ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctxShutDown); err != nil {
				fatalf_stderr("Server forced to shutdown: %s", err)
			}

			log.Println("Server exiting properly")
		}(r)

		// Wait for the main context to be canceled
		for {
			select {
			case <-ctx.Done():
				return
			}
		}
	})
}

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
	name := c.Param("name")

	filePath := fmt.Sprintf("bundled/assets/%v", name)

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
