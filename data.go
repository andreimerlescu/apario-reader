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
	crypto_rand "crypto/rand"
	`crypto/tls`
	"fmt"
	`html/template`
	"image/color"
	`math/big`
	"path/filepath"
	`regexp`
	`sync`
	"sync/atomic"
	"time"

	`github.com/andreimerlescu/configurable`
	cwg `github.com/andreimerlescu/go-countable-waitgroup`
	sch `github.com/andreimerlescu/go-smartchan`
	`github.com/gin-gonic/gin`

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
	m_collections  = make(map[string]Collection)
	mu_collections = sync.RWMutex{}

	m_collection_documents  = make(map[string]map[string]Document) // map[CollectionName][DocumentIdentifier]Document{}
	mu_collection_documents = sync.RWMutex{}

	m_word_pages  = make(map[string]map[string]struct{}) // map[word]map[PageIdentifier]struct{}
	mu_word_pages = sync.RWMutex{}

	m_document_page_number_page  = make(map[string]map[uint]Page) // map[DocumentIdentifier][PageNumber]Page{}
	mu_document_page_number_page = sync.RWMutex{}

	m_document_page_identifiers_pgno  = make(map[string]map[string]uint) // map[DocumentIdentifier]map[PageIdentifier]PageNumber
	mu_document_page_identifiers_pgno = sync.RWMutex{}

	m_document_pgno_page_identifier  = make(map[string]map[uint]string) // map[DocumentIdentifier]map[PageNumber]PageIdentifier
	mu_document_pgno_page_identifier = sync.RWMutex{}

	m_page_identifier_document  = make(map[string]string) // map[PageIdentifier]DocumentIdentifier
	mu_page_identifier_document = sync.RWMutex{}

	m_page_identifier_page_number  = make(map[string]uint) // map[PageIdentifier]PageNumber
	mu_page_identifier_page_number = sync.RWMutex{}

	m_index_page_identifier  = make(map[int64]string) // map[Index]PageIdentifier
	mu_index_page_identifier = sync.RWMutex{}

	m_index_document_identifier  = make(map[int64]string) // map[Index]DocumentIdentifier
	mu_index_document_identifier = sync.RWMutex{}

	m_document_identifier_directory  = make(map[string]string) // map[DocumentIdentifier]checksum inside *flag_s_database
	mu_document_identifier_directory = sync.RWMutex{}

	m_document_identifier_cover_page_identifier  = make(map[string]string) // map[DocumentIdentifier]PageIdentifier
	mu_document_identifier_cover_page_identifier = sync.RWMutex{}

	// TODO: maybe i should switch this to map[GemScore.English]map[word]map[PageIdentifier]struct{}
	m_page_gematria_english  = make(map[uint]map[string]map[string]struct{}) // map[GemScore.English]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_english = sync.RWMutex{}

	// TODO: maybe i should switch this to map[GemScore.Jewish]map[word]map[PageIdentifier]struct{}
	m_page_gematria_jewish  = make(map[uint]map[string]map[string]struct{}) // map[GemScore.Jewish]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_jewish = sync.RWMutex{}

	// TODO: maybe i should switch this to map[GemScore.Simple]map[word]map[PageIdentifier]struct{}
	m_page_gematria_simple  = make(map[uint]map[string]map[string]struct{}) // map[GemScore.Simple]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_simple = sync.RWMutex{}

	m_location_cities  []Location
	mu_location_cities = sync.RWMutex{}

	m_location_countries  []Location
	mu_location_countries = sync.RWMutex{}

	m_location_states  []Location
	mu_location_states = sync.RWMutex{}

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

	// Command Line Flags
	flag_s_database                                     = config.NewString("database", "", "apario-contribution rendered database directory path")
	flag_i_sem_limiter                                  = config.NewInt("limit", channel_buffer_size, "general purpose semaphore limiter")
	flag_i_sem_directories                              = config.NewInt("directories-limiter", channel_buffer_size, "concurrent directories to process out of the database (example: 369)")
	flag_i_sem_pages                                    = config.NewInt("pages-limiter", channel_buffer_size, "concurrent pages to process out of the database")
	flag_i_directory_buffer                             = config.NewInt("directory-buffer", channel_buffer_size, "buffered channel size for pending directories from the database (3x --directories, example: 1107)")
	flag_i_buffer                                       = config.NewInt("buffer", reader_buffer_bytes, "Memory allocation for CSV buffer (min 168 * 1024 = 168KB)")
	flag_g_log_file                                     = config.NewString("log", filepath.Join(".", "logs", fmt.Sprintf("badbitchreads-%04d-%02d-%02d-%02d-%02d-%02d.log", startedAt.Year(), startedAt.Month(), startedAt.Day(), startedAt.Hour(), startedAt.Minute(), startedAt.Second())), "File to save logs to. Default is logs/engine-YYYY-MM-DD-HH-MM-SS.log")
	flag_b_enable_cors                                  = config.NewBool("enable-cors", true, "Enable/Disable CORS")
	flag_b_enable_csp                                   = config.NewBool("enable-csp", true, "Enable/Disable CSP")
	flag_s_cors_domains_csv                             = config.NewString("cors-domains-csv", "", "List of CORS domains in CSV format")
	flag_s_csp_domains_csv                              = config.NewString("csp-domains-csv", "", "List of CSP domains in CSV format")
	flag_s_csp_thirdparty_csv                           = config.NewString("csp-thirdparty-csv", "", "List of third party domains in CSV format")
	flag_s_csp_thirdparty_styles_csv                    = config.NewString("csp-thirdparty-styles-csv", "", "List of third party domains in CSV format")
	flag_s_csp_websocket_domains_csv                    = config.NewString("csp-ws-domains-csv", "", "List of Web Socket domains in CSV format")
	flag_b_csp_script_enable_unsafe_inline              = config.NewBool("csp-script-unsafe-inline", true, "Enable/Disable Unsafe Inline script execution via CSP")
	flag_b_csp_script_enable_unsafe_eval                = config.NewBool("csp-script-unsafe-eval", false, "Enable/Disable Unsafe Eval script execution via CSP")
	flag_b_csp_child_src_enable_unsafe_inline           = config.NewBool("csp-child-unsafe-inline", true, "Enable/Disable Child SRC Unsafe Inline script execution via CSP")
	flag_b_csp_style_src_enable_unsafe_inline           = config.NewBool("csp-style-unsafe-inline", true, "Enable/Disable Style SRC Unsafe Inline script execution via CSP")
	flag_b_csp_upgrade_unsecure_requests                = config.NewBool("csp-upgrade-insecure", true, "Enable/Disable automagically upgrading HTTP to HTTPS for requests via CSP")
	flag_b_csp_block_mixed_content                      = config.NewBool("csp-block-mixed-content", true, "Enable/Disable automatically blocking mixed HTTP and HTTPS content for requests via CSP")
	flag_s_csp_report_uri                               = config.NewString("csp-report-uri", "/security/csp-report", "Path for content security policy violation reports to get logged")
	flag_s_config_file                                  = config.NewString("config", filepath.Join(".", "config.yaml"), "Configuration file")
	flag_i_concurrent_searches                          = config.NewInt("concurrent-searches", 30, "maximum number of allowed concurrent searches before a waiting room appears")
	flag_s_search_algorithm                             = config.NewString("search-algorithm", "jarrow_winkler", "values are wagner_fisher, ukkonen, jaro, jaro_winkler, soundex, hamming ; default is jaro_winkler")
	flag_i_search_concurrency_buffer                    = config.NewInt("search-concurrency-buffer", 369, "buffer channel size for search results ; default = 369")
	flag_i_search_concurrency_limiter                   = config.NewInt("search-concurrency-limiter", 9, "concurrent keyword processing per search query ; default = 9")
	flag_i_search_timeout_seconds                       = config.NewInt("search-timeout-seconds", 30, "maximum seconds to spend on a search")
	flag_f_search_jaro_threshold                        = config.NewFloat64("search-threshold-jaro", 0.71, "1.0 means exact match 0.0 means no match; default is 0.71")
	flag_f_search_jaro_winkler_threshold                = config.NewFloat64("search-threshold-jaro-winkler", 0.71, "using the JaroWinkler method, define the threshold that is tolerated; default is 0.71")
	flag_f_search_jaro_winkler_boost_threshold          = config.NewFloat64("search-jaro-winkler-boost-threshold", 0.7, "weight applied to common prefixes in matched strings comparing dictionary terms, page word data, and search query params")
	flag_i_search_jaro_winkler_prefix_size              = config.NewInt("search-jaro-winkler-prefix-size", 3, "length of a jarrow weighted prefix string")
	flag_i_search_ukkonen_icost                         = config.NewInt("search-ukkonen-icost", 1, "insert cost ; when adding a char to find a match ; increase the score by this number ; default = 1")
	flag_i_search_ukkonen_scost                         = config.NewInt("search-ukkonen-scost", 2, "substitution cost ; when replacing a char increase the score by this number ; default = 2")
	flag_i_search_ukkonen_dcost                         = config.NewInt("search-ukkonen-dcost", 1, "delete cost ; when removing a char to find a match ; increase the score by this number ; default = 1")
	flag_i_search_ukkonen_max_substitutions             = config.NewInt("search-ukkonen-max-substitutions", 2, "maximum number of substitutions allowed for a word to be considered a match ; higher value = lower accurate ; lower value = higher accuracy ; min = 0; default = 2")
	flag_i_search_wagner_fischer_icost                  = config.NewInt("search-wagner-fischer-icost", 1, "insert cost ; when adding a char to find a match ; increase the score by this number ; default = 1")
	flag_i_search_wagner_fischer_scost                  = config.NewInt("search-wagner-fischer-scost", 2, "substitution cost ; when replacing a char increase the score by this number ; default = 2")
	flag_i_search_wagner_fischer_dcost                  = config.NewInt("search-wagner-fischer-dcost", 1, "delete cost ; when removing a char to find a match ; increase the score by this number ; default = 1")
	flag_i_search_wagner_fischer_max_substitutions      = config.NewInt("search-wagner-fischer-max-substitutions", 2, "maximum number of substitutions allowed for a word to be considered a match ; higher value = lower accurate ; lower value = higher accuracy ; min = 0; default = 2")
	flag_i_search_hamming_max_substitutions             = config.NewInt("search-hamming-max-substitutions", 2, "maximum number of substitutions allowed for a word to be considered a match ; higher value = lower accuracy ; min = 1 ; default = 2")
	flag_s_log_file                                     = config.NewString("error-log", filepath.Join(".", "logs", "go.log"), "File to write logs.")
	flag_s_gin_log_file                                 = config.NewString("access-log", filepath.Join(".", "logs", "gin.log"), "Default log file for GIN access logs.")
	flag_i_webserver_default_port                       = config.NewInt("unsecure-port", 8080, "Port to start non-SSL version of application.")
	flag_i_webserver_secure_port                        = config.NewInt("secure-port", 8443, "Port to start the SSL version of the application.")
	flag_s_ssl_public_key                               = config.NewString("tls-public-key", "", "Path to the SSL certificate's public key. It expects any CA chain certificates to be concatenated at the end of this PEM formatted file.")
	flag_s_ssl_private_key                              = config.NewString("tls-private-key", "", "Path to the PEM formatted SSL certificate's private key.")
	flag_s_ssl_private_key_password                     = config.NewString("tls-private-key-password", "", "If the PEM private key is encrypted with a password, provide it here.")
	flag_b_auto_ssl                                     = config.NewBool("auto-tls", false, "Create a self-signed certificate on the fly and use it for serving the application over SSL.")
	flag_i_reload_cert_every_minutes                    = config.NewInt("tls-life-min", 72, "Lifespan of the auto generated self signed TLS certificate in minutes.")
	flag_i_auto_ssl_default_expires                     = config.NewInt("tls-expires-in", 365*24, "Auto generated TLS/SSL certificates will automatically expire in hours.")
	flag_s_auto_ssl_company                             = config.NewString("tls-company", "ACME Inc.", "Auto generated TLS/SSL certificates are configured with the company name.")
	flag_s_auto_ssl_domain_name                         = config.NewString("tls-domain-name", "", "Auto generated TLS/SSL certificates will have this common name and run on this domain name.")
	flag_s_auto_ssl_san_ip                              = config.NewString("tls-san-ip", "", "Auto generated TLS/SSL certificates will have this SAN IP address attached to it in addition to its common name.")
	flag_s_auto_ssl_additional_domains                  = config.NewString("tls-additional-domains", "", "Auto generated TLS/SSL certificates will be issued with these additional domains (CSV formatted).")
	flag_f_rate_limit                                   = config.NewFloat64("rate-limit", 12.0, "Requests per second (0.5 = 1 request every 2 seconds).")
	flag_i_rate_limit_cleanup_delay                     = config.NewInt("rate-limit-cleanup", 3, "Seconds between rate limit cleanups.")
	flag_i_rate_limit_entry_ttl                         = config.NewInt("rate-limit-ttl", 3, "Seconds a rate limit entry exists for before cleanup is triggered.")
	flag_f_asset_rate_limit                             = config.NewFloat64("rate-limit-asset", 36.0, "Requests per second (0.5 = 1 request every 2 seconds).")
	flag_i_asset_rate_limit_cleanup_delay               = config.NewInt("rate-limit-asset-cleanup", 17, "Seconds between rate limit cleanups.")
	flag_i_asset_rate_limit_entry_ttl                   = config.NewInt("rate-limit-asset-ttl", 17, "Seconds a rate limit entry exists for before cleanup is triggered.")
	flag_s_trusted_proxies                              = config.NewString("trusted-proxies", "", "Configure the web server to forward client IP addresses to the application if a proxy is used such as Nginx; set that proxy's IP here.")
	flag_s_site_title                                   = config.NewString("site-title", "Project Apario", "title of the application that appears on the web gui")
	flag_s_site_company                                 = config.NewString("company-name", "Project Apario LLC", "name of the company that operates the service")
	flag_s_primary_domain                               = config.NewString("primary-domain", "projectapario.com", "primary domain name used to access the service")
	flag_s_number_decimal_place                         = config.NewString("decimal-symbol", ",", "symbol for decimals, default is .")
	flag_s_dark_mode_cookie                             = config.NewString("dark-mode-cookie-name", "dark-mode", "set the name of the cookie for dark mode")
	flag_b_use_cookies                                  = config.NewBool("use-cookies", true, "toggle using cookies or not - cookies and sessions can be true but both cannot be false")
	flag_s_cookie_domain                                = config.NewString("cookie-domain", "localhost:8080", "domain to use for cookies")
	flag_b_use_sessions                                 = config.NewBool("use-sessions", false, "toggle using sessions or not - cookies and sessions can be true but both cannot be false")
	flag_s_session_store                                = config.NewString("session-store", "cookie", "where to store sessions - choices are cookie or redis ; cannot be cookie if use-cookies is false")
	flag_i_session_store_redis_connections              = config.NewInt("session-store-redis-connections", 10, "number of connections to maintain with redis between this application")
	flag_s_session_store_redis_protocol                 = config.NewString("session-store-redis-protocol", "tcp", "how the connection to redis is established - default is tcp")
	flag_s_session_store_redis_password                 = config.NewString("session-store-redis-password", "", "password configured in redis that this app will use to communicate")
	flag_i_session_store_redis_database                 = config.NewInt("session-store-redis-database", 3, "the database ID in redis that sessions will be stored ; default is 3")
	flag_i_session_store_redis_tls_insecure_skip_verify = config.NewBool("session-store-redis-tls-insecure-skip-verify", false, "false enforces tls certification and true disables tls verification")
	flag_b_session_store_redis_tls_enabled              = config.NewBool("session-store-redis-tls-enabled", false, "is tls encryption enabled on the redis server? default is false")
	flag_s_session_store_redis_tls_certificate_path     = config.NewString("session-store-redis-tls-certificate-path", "", "where is the tls certificate for redis? (pem format required)")
	flag_s_session_store_redis_tls_private_key_path     = config.NewString("session-store-redis-tls-private-key-path", "", "where is the private key for redis? (pem format required)")
	flag_s_session_store_redis_tls_root_ca_path         = config.NewString("session-store-redis-tls-root-ca-path", "", "where is the root ca certificate bundle for redis? (pem format required)")
	flag_s_session_store_redis_servers                  = config.NewString("session-store-redis-servers", "localhost:6379", "comma separated list of redis servers. example: '10.0.0.2:6379,10.0.0.3:6379,10.0.0.4:6379' default: 'localhost:6379'")
	flag_b_session_store_redis_fallback_cookie          = config.NewBool("session-store-redis-fallback-cookie", false, "fall back to use cookies if and when redis is temporarily unavailable")
	flag_s_session_store_cookie_secret                  = config.NewString("session-store-cookie-secret", "secure-password-369-goes-here", "a password to secure the cookies")
	flag_s_session_store_redis_secret                   = config.NewString("session-store-redis-secret", "secure-password-369-goes-here", "a password to secure the redis sessions")

	// Atomics
	a_b_gematria_loaded   = atomic.Bool{}
	a_b_cryptonyms_loaded = atomic.Bool{}
	a_b_locations_loaded  = atomic.Bool{}
	a_i_total_documents   = atomic.Int64{}
	a_i_total_pages       = atomic.Int64{}
	a_i_waiting_room      = atomic.Int64{}

	// Semaphores
	sem_db_directories      = sema.New(*flag_i_sem_directories)
	sem_analyze_pages       = sema.New(*flag_i_sem_pages)
	sem_concurrent_searches = sema.New(*flag_i_concurrent_searches)

	// Channels
	ch_db_directories       = sch.NewSmartChan(*flag_i_directory_buffer)
	ch_cert_reloader_cancel = sch.NewSmartChan(1)
	ch_webserver_done       = sch.NewSmartChan(1)

	// gin templates
	gin_func_map          template.FuncMap
	default_gin_func_vars gin.H
	// If a new template is needed, and you're going to use it in a new route entry point, make sure that
	// you define your variables either here at compile time OR within your function. If you choose to
	// define them inside your function before you invoke your template, make sure that you use the
	// mu_gin_func_vars mutex to ensure proper locking/unlocking for read/write.
	gin_func_vars    map[string]gin.H
	mu_gin_func_vars = sync.RWMutex{}
)

type SearchResult struct {
	Query    string         `json:"query"`
	Analysis SearchAnalysis `json:"search_analysis"`
	Total    int            `json:"total_results"`
	//Inclusive []string       `json:"inclusive_identifiers"`
	//Exclusive []string       `json:"exclusive_identifiers"`
	Results []string `json:"page_identifiers"`
}

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
	Identifier         string   `json:"identifier"`
	DocumentIdentifier string   `json:"document_identifier"`
	FullText           string   `json:"full_text"`
	PageNumber         uint     `json:"page_number"`
	Gematria           GemScore `json:"gematria"`
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

type SearchAnalysis struct {
	Ors  map[uint]string
	Ands []string
	Nots []string
}

func f_i_random_int(limit int) int {
	if limit <= 0 {
		return 0
	}

	newInt, err := crypto_rand.Int(crypto_rand.Reader, big.NewInt(int64(limit)))
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}
	return int(newInt.Int64())
}

// f_i_random_int_range return a random int that is between start and limit, will recursively run until match found
func f_i_random_int_range(start int, limit int) int {
	i := f_i_random_int(limit)
	if i >= start && i <= limit {
		return i
	} else {
		return f_i_random_int_range(start, limit) // retry
	}
}

func f_i_plus(a, b int) int {
	return a + b
}

func f_i_minus(a, b int) int {
	return a - b
}

func f_i_sequence(start, end int) []int {
	var seq []int
	for i := start; i <= end; i++ {
		seq = append(seq, i)
	}
	return seq
}

func f_i_max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func f_i_min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
