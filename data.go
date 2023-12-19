/*
Project Apario is the World's Truth Repository that was invented and started by Andrei Merlescu in 2020.
Copyright (C) 2023  Andrei Merlescu

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package main

import (
	`context`
	`crypto/tls`
	"fmt"
	"image/color"
	"path/filepath"
	`regexp`
	`sync`
	"sync/atomic"
	"time"

	`github.com/andreimerlescu/configurable`
	cwg `github.com/andreimerlescu/go-countable-waitgroup`
	sch `github.com/andreimerlescu/go-smartchan`

	`badbitchreads/sema`
)

const (
	c_retry_attempts     = 33
	c_identifier_charset = "ABCDEFGHKMNPQRSTUVWXYZ123456789"
	c_dir_permissions    = 0111
)

var (
	startedAt = time.Now().UTC()
	config    = configurable.New()

	// Integers
	channel_buffer_size    int = 1          // Buffered Channel's Size
	reader_buffer_bytes    int = 128 * 1024 // 128KB default buffer for reading CSV, XLSX, and PSV files into memory
	jpeg_compression_ratio     = 90         // Progressive JPEG Quality (valid options are 1-100)

	// Colors
	color_background = color.RGBA{R: 40, G: 40, B: 86, A: 255}    // navy blue
	color_text       = color.RGBA{R: 250, G: 226, B: 203, A: 255} // sky yellow

	// Strings

	// Maps
	m_cryptonyms = make(map[string]string) // map[Cryptonym]Definition
	//m_words                          = make(map[string]map[string]struct{})          // map[language]map[word]{}
	//m_words_english_gematria_english = make(map[string]uint)
	//m_words_english_gematria_jewish  = make(map[string]uint)
	//m_words_english_gematria_simple  = make(map[string]uint)
	//m_gematria_english               = make(map[uint]map[string]struct{})            // english words gematria english values
	//m_gematria_jewish                = make(map[uint]map[string]struct{})            // english words gematria jewish values
	//m_gematria_simple                = make(map[uint]map[string]struct{})            // english words gematria simple values
	m_collections            = make(map[string]Collection)
	mu_collections           = sync.RWMutex{}
	m_collection_documents   = make(map[string]map[string]Document)          // map[CollectionName][DocumentIdentifier]Document{}
	mu_collection_documents  = sync.RWMutex{}
	m_document_pages         = make(map[string]map[uint]Page)                // map[DocumentIdentifier][PageNumber]Page{}
	mu_document_pages        = sync.RWMutex{}
	m_page_words             = make(map[string]map[string]struct{})          // map[PageIdentifier]map[word]struct{}
	mu_page_words            = sync.RWMutex{}
	m_page_gematria_english  = make(map[uint]map[string]map[string]struct{}) // map[GemScore.English]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_english = sync.RWMutex{}
	m_page_gematria_jewish   = make(map[uint]map[string]map[string]struct{}) // map[GemScore.Jewish]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_jewish  = sync.RWMutex{}
	m_page_gematria_simple   = make(map[uint]map[string]map[string]struct{}) // map[GemScore.Simple]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_simple  = sync.RWMutex{}

	m_location_cities     []Location
	mu_location_cities    = sync.RWMutex{}
	m_location_countries  []Location
	mu_location_countries = sync.RWMutex{}
	m_location_states     []Location
	mu_location_states    = sync.RWMutex{}

	// old maps
	m_gcm_jewish  = make(GemCodeMap)
	m_gcm_english = make(GemCodeMap)
	m_gcm_simple  = make(GemCodeMap)
	m_months      = map[string]time.Month{
		"jan": time.January, "january": time.January, "01": time.January, "1": time.January,
		"feb": time.February, "february": time.February, "02": time.February, "2": time.February,
		"mar": time.March, "march": time.March, "03": time.March, "3": time.March,
		"apr": time.April, "april": time.April, "04": time.April, "4": time.April,
		"may": time.May, "05": time.May, "5": time.May,
		"jun": time.June, "june": time.June, "06": time.June, "6": time.June,
		"jul": time.July, "july": time.July, "07": time.July, "7": time.July,
		"aug": time.August, "august": time.August, "08": time.August, "8": time.August,
		"sep": time.September, "september": time.September, "09": time.September, "9": time.September,
		"oct": time.October, "october": time.October, "10": time.October,
		"nov": time.November, "november": time.November, "11": time.November,
		"dec": time.December, "december": time.December, "12": time.December,
	}

	// Regex
	re_date1 = regexp.MustCompile(`(?i)(\d{1,2})(st|nd|rd|th)?\s(?:of\s)?(January|Jan|February|Feb|March|Mar|April|Apr|May|June|Jun|July|Jul|August|Aug|September|Sep|October|Oct|November|Nov|December|Dec),?\s(\d{2,4})`)
	re_date2 = regexp.MustCompile(`(?i)(\d{1,2})\/(\d{1,2})\/(\d{2,4})`)
	re_date3 = regexp.MustCompile(`(?i)(January|Jan|February|Feb|March|Mar|April|Apr|May|June|Jun|July|Jul|August|Aug|September|Sep|October|Oct|November|Nov|December|Dec),?\s(\d{2,4})`)
	re_date5 = regexp.MustCompile(`(?i)(January|Jan|February|Feb|March|Mar|April|Apr|May|June|Jun|July|Jul|August|Aug|September|Sep|October|Oct|November|Nov|December|Dec)\s(\d{1,2})(st|nd|rd|th)?,?\s(\d{2,4})`)
	re_date4 = regexp.MustCompile(`(?i)(January|Jan|February|Feb|March|Mar|April|Apr|May|June|Jun|July|Jul|August|Aug|September|Sep|October|Oct|November|Nov|December|Dec)\s(\d{4})`)
	re_date6 = regexp.MustCompile(`(\d{4})`)

	// Synchronization
	wg_active_tasks   = cwg.CountableWaitGroup{}
	mu_cert           = &sync.RWMutex{}
	once_server_start = sync.Once{}

	// TLS
	cert tls.Certificate

	// Channels
	ch_cert_reloader_cancel = make(chan bool)
	ch_webserver_done       = make(chan struct{})

	// Command Line Flags
	flag_s_database                           = config.NewString("database", "", "apario-contribution rendered database directory path")
	flag_i_sem_limiter                        = config.NewInt("limit", channel_buffer_size, "general purpose semaphore limiter")
	flag_i_sem_directories                    = config.NewInt("directories-limiter", channel_buffer_size, "concurrent directories to process out of the database (example: 369)")
	flag_i_sem_pages                          = config.NewInt("pages-limiter", channel_buffer_size, "concurrent pages to process out of the database")
	flag_i_directory_buffer                   = config.NewInt("directory-buffer", channel_buffer_size, "buffered channel size for pending directories from the database (3x --directories, example: 1107)")
	flag_i_buffer                             = config.NewInt("buffer", reader_buffer_bytes, "Memory allocation for CSV buffer (min 168 * 1024 = 168KB)")
	flag_g_log_file                           = config.NewString("log", filepath.Join(".", "logs", fmt.Sprintf("badbitchreads-%04d-%02d-%02d-%02d-%02d-%02d.log", startedAt.Year(), startedAt.Month(), startedAt.Day(), startedAt.Hour(), startedAt.Minute(), startedAt.Second())), "File to save logs to. Default is logs/engine-YYYY-MM-DD-HH-MM-SS.log")
	flag_b_enable_cors                        = config.NewBool("enable-cors", true, "Enable/Disable CORS")
	flag_b_enable_csp                         = config.NewBool("enable-csp", true, "Enable/Disable CSP")
	flag_s_cors_domains_csv                   = config.NewString("cors-domains-csv", "", "List of CORS domains in CSV format")
	flag_s_csp_domains_csv                    = config.NewString("csp-domains-csv", "", "List of CSP domains in CSV format")
	flag_s_csp_thirdparty_csv                 = config.NewString("csp-thirdparty-csv", "", "List of third party domains in CSV format")
	flag_s_csp_thirdparty_styles_csv          = config.NewString("csp-thirdparty-styles-csv", "", "List of third party domains in CSV format")
	flag_s_csp_websocket_domains_csv          = config.NewString("csp-ws-domains-csv", "", "List of Web Socket domains in CSV format")
	flag_b_csp_script_enable_unsafe_inline    = config.NewBool("csp-script-unsafe-inline", true, "Enable/Disable Unsafe Inline script execution via CSP")
	flag_b_csp_script_enable_unsafe_eval      = config.NewBool("csp-script-unsafe-eval", false, "Enable/Disable Unsafe Eval script execution via CSP")
	flag_b_csp_child_src_enable_unsafe_inline = config.NewBool("csp-child-unsafe-inline", true, "Enable/Disable Child SRC Unsafe Inline script execution via CSP")
	flag_b_csp_style_src_enable_unsafe_inline = config.NewBool("csp-style-unsafe-inline", true, "Enable/Disable Style SRC Unsafe Inline script execution via CSP")
	flag_b_csp_upgrade_unsecure_requests      = config.NewBool("csp-upgrade-insecure", true, "Enable/Disable automagically upgrading HTTP to HTTPS for requests via CSP")
	flag_b_csp_block_mixed_content            = config.NewBool("csp-block-mixed-content", true, "Enable/Disable automatically blocking mixed HTTP and HTTPS content for requests via CSP")
	flag_s_csp_report_uri                     = config.NewString("csp-report-uri", "/security/csp-report", "Path for content security policy violation reports to get logged")
	flag_s_config_file                        = config.NewString("config", filepath.Join(".", "config.yaml"), "Configuration file")
	flag_s_log_file                           = config.NewString("error-log", filepath.Join(".", "logs", "go.log"), "File to write logs.")
	flag_s_gin_log_file                       = config.NewString("access-log", filepath.Join(".", "logs", "gin.log"), "Default log file for GIN access logs.")
	flag_i_webserver_default_port             = config.NewInt("unsecure-port", 8080, "Port to start non-SSL version of application.")
	flag_i_webserver_secure_port              = config.NewInt("secure-port", 8443, "Port to start the SSL version of the application.")
	flag_s_ssl_public_key                     = config.NewString("tls-public-key", "", "Path to the SSL certificate's public key. It expects any CA chain certificates to be concatenated at the end of this PEM formatted file.")
	flag_s_ssl_private_key                    = config.NewString("tls-private-key", "", "Path to the PEM formatted SSL certificate's private key.")
	flag_s_ssl_private_key_password           = config.NewString("tls-private-key-password", "", "If the PEM private key is encrypted with a password, provide it here.")
	flag_b_auto_ssl                           = config.NewBool("auto-tls", false, "Create a self-signed certificate on the fly and use it for serving the application over SSL.")
	flag_i_reload_cert_every_minutes          = config.NewInt("tls-life-min", 72, "Lifespan of the auto generated self signed TLS certificate in minutes.")
	flag_i_auto_ssl_default_expires           = config.NewInt("tls-expires-in", 365*24, "Auto generated TLS/SSL certificates will automatically expire in hours.")
	flag_s_auto_ssl_company                   = config.NewString("tls-company", "ACME Inc.", "Auto generated TLS/SSL certificates are configured with the company name.")
	flag_s_auto_ssl_domain_name               = config.NewString("tls-domain-name", "", "Auto generated TLS/SSL certificates will have this common name and run on this domain name.")
	flag_s_auto_ssl_san_ip                    = config.NewString("tls-san-ip", "", "Auto generated TLS/SSL certificates will have this SAN IP address attached to it in addition to its common name.")
	flag_s_auto_ssl_additional_domains        = config.NewString("tls-additional-domains", "", "Auto generated TLS/SSL certificates will be issued with these additional domains (CSV formatted).")
	flag_f_rate_limit                         = config.NewFloat64("rate-limit", 12.0, "Requests per second (0.5 = 1 request every 2 seconds).")
	flag_i_rate_limit_cleanup_delay           = config.NewInt("rate-limit-cleanup", 3, "Seconds between rate limit cleanups.")
	flag_i_rate_limit_entry_ttl               = config.NewInt("rate-limit-ttl", 3, "Seconds a rate limit entry exists for before cleanup is triggered.")
	flag_f_asset_rate_limit                   = config.NewFloat64("rate-limit-asset", 36.0, "Requests per second (0.5 = 1 request every 2 seconds).")
	flag_i_asset_rate_limit_cleanup_delay     = config.NewInt("rate-limit-asset-cleanup", 17, "Seconds between rate limit cleanups.")
	flag_i_asset_rate_limit_entry_ttl         = config.NewInt("rate-limit-asset-ttl", 17, "Seconds a rate limit entry exists for before cleanup is triggered.")
	flag_s_trusted_proxies                    = config.NewString("trusted-proxies", "", "Configure the web server to forward client IP addresses to the application if a proxy is used such as Nginx; set that proxy's IP here.")

	// Atomics
	a_b_gematria_loaded   = atomic.Bool{}
	a_b_cryptonyms_loaded = atomic.Bool{}
	a_b_locations_loaded  = atomic.Bool{}

	// Semaphores
	sem_db_directories = sema.New(*flag_i_sem_directories)
	sem_analyze_pages  = sema.New(*flag_i_sem_pages)

	// Channels
	ch_db_directories = sch.NewSmartChan(*flag_i_directory_buffer)
)

type CtxKey string
type CallbackFunc func(ctx context.Context, row []Column) error

type GemCodeMap map[string]uint

type GemScore struct {
	Jewish  uint
	English uint
	Simple  uint
}

type Geography struct {
	Countries []CountableLocation `json:"countries"`
	States    []CountableLocation `json:"states"`
	Cities    []CountableLocation `json:"cities"`
}

type CountableLocation struct {
	Location *Location `json:"location"`
	Quantity int       `json:"quantity"`
}

type Location struct {
	Continent   string  `json:"continent"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
}

type Collection struct {
	Name      string              `json:"name"`
	Documents map[string]Document `json:"documents"`
}

type Document struct {
	Identifier   string            `json:"identifier"`
	RecordNumber string            `json:"record_number"`
	Pages        map[uint]Page     `json:"pages"`
	Metadata     map[string]string `json:"metadata"`
	TotalPages   uint              `json:"total_pages"`
	Hyperlink    string            `json:"url"`
	DatabasePath string            `json:"database_path"`
}

type Page struct {
	FullText   string   `json:"full_text"`
	PageNumber uint     `json:"page_number"`
	Identifier string   `json:"identifier"`
	Gematria   GemScore `json:"gematria"`
}

type Gematria struct {
	Word  string   `json:"word"`
	Score GemScore `json:"score"`
}

type WordResult struct {
	Word     string   `json:"word"`
	Language string   `json:"language"`
	Gematria Gematria `json:"gematria"`
	Quantity int      `json:"quantity"`
}

type ResultData struct {
	Identifier        string            `json:"identifier"`
	URL               string            `json:"url"`
	DataDir           string            `json:"data_dir"`
	PDFPath           string            `json:"pdf_path"`
	PDFChecksum       string            `json:"pdf_checksum"`
	OCRTextPath       string            `json:"ocr_text_path"`
	ExtractedTextPath string            `json:"extracted_text_path"`
	RecordPath        string            `json:"record_path"`
	TotalPages        int64             `json:"total_pages"`
	Metadata          map[string]string `json:"metadata"`
}

type PendingPage struct {
	Identifier       string              `json:"identifier"`
	RecordIdentifier string              `json:"record_identifier"`
	PageNumber       int                 `json:"page_number"`
	PDFPath          string              `json:"pdf_path"`
	PagesDir         string              `json:"pages_dir"`
	OCRTextPath      string              `json:"ocr_text_path"`
	ManifestPath     string              `json:"manifest_path"`
	Language         string              `json:"language"`
	Words            []WordResult        `json:"words"`
	Cryptonyms       []string            `json:"cryptonyms"`
	Dates            []time.Time         `json:"dates"`
	Geography        Geography           `json:"geography"`
	Gematrias        map[string]Gematria `json:"gematrias"`
	JPEG             JPEG                `json:"jpeg"`
	PNG              PNG                 `json:"png"`
}

type JPEG struct {
	Light Images `json:"light"`
	Dark  Images `json:"dark"`
}

type PNG struct {
	Light Images `json:"light"`
	Dark  Images `json:"dark"`
}

type Images struct {
	Original string `json:"original"`
	Large    string `json:"large"`
	Medium   string `json:"medium"`
	Small    string `json:"small"`
	Social   string `json:"social"`
}

type Column struct {
	Header string
	Value  string
}
