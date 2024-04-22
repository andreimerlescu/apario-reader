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
	`html/template`
	"image/color"
	`net`
	`regexp`
	`sync`
	"sync/atomic"
	"time"

	ai `github.com/andreimerlescu/go-apario-identifier`
	cwg `github.com/andreimerlescu/go-countable-waitgroup`
	sema `github.com/andreimerlescu/go-sema`
	sch `github.com/andreimerlescu/go-smartchan`
	`github.com/gin-gonic/gin`
	`github.com/gorilla/sessions`
)

const (
	c_retry_attempts     = 33
	c_identifier_charset = "ABCDEFGHKMNPQRSTUVWXYZ123456789"
	c_dir_permissions    = 0111
)

var (
	startedAt = time.Now().UTC()

	valet_db_data     = ai.NewValet(*flag_s_database)
	valet_db_users    = ai.NewValet(*flag_s_users_database_directory)
	valet_db_snippets = ai.NewValet(*flag_s_snippets_database)
	valet_db_textee   = ai.NewValet(*flag_s_textee_database_path)
	valet_db_tags     = ai.NewValet(*flag_s_tag_database_path)

	session_store = sessions.NewFilesystemStore(*flag_s_sessions_directory, []byte(*flag_s_session_secret))

	// Integers
	channel_buffer_size    int = 1          // Buffered Channel's Size
	reader_buffer_bytes    int = 128 * 1024 // 128KB default buffer for reading CSV, XLSX, and PSV files into memory
	jpeg_compression_ratio     = 90         // Progressive JPEG Quality (valid options are 1-100)

	// Colors
	color_background = color.RGBA{R: 40, G: 40, B: 86, A: 255}    // navy blue
	color_text       = color.RGBA{R: 250, G: 226, B: 203, A: 255} // sky yellow

	// Maps
	m_words  = make(map[string]map[string]struct{}) // map[language]map[word]{}
	mu_words = sync.RWMutex{}

	m_words_english_gematria_english  = make(map[string]uint)
	mu_words_english_gematria_english = sync.RWMutex{}

	m_words_english_gematria_jewish  = make(map[string]uint)
	mu_words_english_gematria_jewish = sync.RWMutex{}

	m_words_english_gematria_simple  = make(map[string]uint)
	mu_words_english_gematria_simple = sync.RWMutex{}

	m_gematria_english  = make(map[uint]map[string]struct{}) // english words gematria english values
	mu_gematria_english = sync.RWMutex{}

	m_gematria_jewish  = make(map[uint]map[string]struct{}) // english words gematria jewish values
	mu_gematria_jewish = sync.RWMutex{}

	m_gematria_simple  = make(map[uint]map[string]struct{}) // english words gematria simple values
	mu_gematria_simple = sync.RWMutex{}

	m_cryptonyms  = make(map[string]string) // map[Cryptonym]Definition
	mu_cryptonyms = sync.RWMutex{}

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

	m_document_total_pages  = make(map[string]uint) // map[DocumentIdentifier]TotalPages
	mu_document_total_pages = sync.RWMutex{}

	m_document_source_url  = make(map[string]string) // map[DocumentIdentifier]URL
	mu_document_source_url = sync.RWMutex{}

	m_document_metadata  = make(map[string]map[string]string) // map[DocumentIdentifier]map[Key]Value
	mu_document_metadata = sync.RWMutex{}

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

	// Security
	mu_ip_watch_list = &sync.RWMutex{}
	m_ip_watch_list  = map[string]*atomic.Int64{}

	mu_ip_ban_list          = &sync.RWMutex{}
	m_ip_ban_list  []net.IP = []net.IP{}

	m_online_list  = map[string]online_entry{} // map[ip]online_entry{}
	mu_online_list = sync.RWMutex{}

	a_i_cached_online_counter = atomic.Int64{}
	sem_banned_ip_patch       = sema.New(1)

	sem_concurrent_crypt_actions                = sema.New(*flag_s_session_concurrent_crypt_actions)
	sem_identifier_generator_concurrency_factor = sema.New(1)

	// Synchronization
	wg_active_tasks   = cwg.CountableWaitGroup{}
	mu_cert           = &sync.RWMutex{}
	mu_db_dump        = &sync.RWMutex{}
	mu_db_restore     = &sync.RWMutex{}
	wg_db_dump        = cwg.CountableWaitGroup{}
	wg_db_restore     = cwg.CountableWaitGroup{}
	once_server_start = sync.Once{}

	// TLS
	cert tls.Certificate

	// Atomics
	a_b_gematria_loaded   = atomic.Bool{}
	a_b_cryptonyms_loaded = atomic.Bool{}
	a_b_locations_loaded  = atomic.Bool{}
	a_i_total_documents   = atomic.Int64{}
	a_i_total_pages       = atomic.Int64{}
	a_i_waiting_room      = atomic.Int64{}
	a_b_database_loaded   = atomic.Bool{}

	// Regular Expressions
	reg_identifier = regexp.MustCompile("[^a-zA-Z0-9]+")
	reg_image_size = regexp.MustCompile(`^[a-zA-Z0-9]+\.[a-zA-Z0-9]+$`)
	reg_pdf_name   = regexp.MustCompile(`^[a-zA-Z0-9._-]+\.pdf$`)

	// Semaphores
	sem_db_directories      = sema.New(*flag_i_sem_directories)
	sem_analyze_pages       = sema.New(*flag_i_sem_pages)
	sem_concurrent_searches = sema.New(*flag_i_concurrent_searches)
	sem_image_views         = sema.New(*flag_i_concurrent_image_views)
	sem_asset_requests      = sema.New(*flag_i_concurrent_asset_requests)
	sem_pdf_downloads       = sema.New(*flag_i_concurrent_pdf_downloads)
	sem_db_dump             = sema.New(1)
	sem_db_restore          = sema.New(1)

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

	s_dictionary_languages = []string{"english", "french", "german", "romanian", "russian", "spanish"}
)

type online_entry struct {
	UserAgent     string        `json:"ua"`
	IP            net.IP        `json:"ip"`
	FirstAction   time.Time     `json:"fa"`
	LastAction    time.Time     `json:"la"`
	Hits          *atomic.Int64 `json:"h"`
	LastPath      string        `json:"lp"`
	Authenticated bool          `json:"au"`
	Administrator bool          `json:"ad"`
	Username      string        `json:"un"`
	Reputation    float64       `json:"r"`
}

type ts_ip_save_entry struct {
	IP      net.IP ` json:"ip"`
	Counter int64  `json:"c"`
}
type ts_ip_save struct {
	Entries map[string]ts_ip_save_entry `json:"e"`
	mu      *sync.RWMutex
}

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
	Identifier         string `json:"identifier"`
	DocumentIdentifier string `json:"document_identifier"`
	PageNumber         uint   `json:"page_number"`
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
	Identifier       string      `json:"identifier"`
	RecordIdentifier string      `json:"record_identifier"`
	PageNumber       int         `json:"page_number"`
	PDFPath          string      `json:"pdf_path"`
	PagesDir         string      `json:"pages_dir"`
	OCRTextPath      string      `json:"ocr_text_path"`
	ManifestPath     string      `json:"manifest_path"`
	Language         string      `json:"language"`
	Cryptonyms       []string    `json:"cryptonyms"`
	Dates            []time.Time `json:"dates"`
	JPEG             JPEG        `json:"jpeg"`
	PNG              PNG         `json:"png"`
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
	Header string `json:"h"`
	Value  string `json:"v"`
}

type SearchAnalysis struct {
	Ors  map[uint]string `json:"o"`
	Ands []string        `json:"a"`
	Nots []string        `json:"n"`
}

const (
	kilobyte = 1024
	megabyte = 1024 * kilobyte
	gigabyte = 1024 * megabyte
	terabyte = 1024 * gigabyte
	petabyte = 1024 * terabyte
)

const c_s_default_robots_txt = `User-agent: *
Disallow: /`

const c_s_default_ads_txt = ``

const c_s_default_security_txt = ``

const svg_page_loading_img_src string = `data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4KPHN2ZyB3aWR0aD0iMjUwcHgiIGhlaWdodD0iMzIycHgiIHZpZXdCb3g9IjAgMCAyNTAgMzIyIiB2ZXJzaW9uPSIxLjEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiPgogICAgPHRpdGxlPkFydGJvYXJkPC90aXRsZT4KICAgIDxnIGlkPSJBcnRib2FyZCIgc3Ryb2tlPSJub25lIiBzdHJva2Utd2lkdGg9IjEiIGZpbGw9Im5vbmUiIGZpbGwtcnVsZT0iZXZlbm9kZCI+CiAgICAgICAgPHJlY3QgaWQ9IlJlY3RhbmdsZSIgZmlsbD0iI0YzRjNGMyIgeD0iMCIgeT0iMCIgd2lkdGg9IjI1MCIgaGVpZ2h0PSIzMjIiPjwvcmVjdD4KICAgICAgICA8cGF0aCBkPSJNNDIuMDA1LDEzNC4wNTMgQzQyLjUsMTMzLjk1NCA0Mi43MzEsMTMzLjU1OCA0Mi44MywxMzMuMjk0IEM0My4xNiwxMzIuNjM0IDQzLjE5MywxMzEuODQyIDQzLjIyNiwxMzEuMDgzIEM0My4yNTksMTI5LjEzNiA0My4yOTIsMTI1LjQwNyA0My4yOTIsMTIzLjQ2IEM0NC4xMTcsMTIzLjY5MSA0NS4xMDcsMTIzLjYyNSA0NS45MzIsMTIzLjQyNyBDNDcuMDIxLDEyMy4xNjMgNDguMDQ0LDEyMi43NjcgNDkuMDM0LDEyMi4yNzIgQzUwLjM4NywxMjEuNTc5IDUxLjY0MSwxMjAuNjU1IDUyLjY5NywxMTkuNDY3IEM1My40ODksMTE4LjU3NiA1NC4zMTQsMTE3LjQ1NCA1NC42MTEsMTE2LjI5OSBDNTQuODA5LDExNS40NDEgNTQuNzQzLDExNC41MTcgNTQuMjgxLDExMy42NTkgQzUzLjc1MywxMTIuNzAyIDUyLjgyOSwxMTIuMDc1IDUxLjgwNiwxMTIuMDA5IEM0OC40MDcsMTExLjgxMSA0NS43MzQsMTEzLjQ5NCA0My40NTcsMTE1LjgzNyBDNDMuNDU3LDExNS40NDEgNDMuNDU3LDExNC45NzkgNDMuNTg5LDExNC41ODMgQzQzLjU4OSwxMTQuNTE3IDQzLjYyMiwxMTQuNDUxIDQzLjY1NSwxMTQuMzg1IEM0My43ODcsMTE0LjEyMSA0My45MTksMTEzLjkyMyA0My45MTksMTEzLjY1OSBDNDMuODg2LDExMy40OTQgNDMuNzg3LDExMy4xOTcgNDMuNTIzLDExMy4wOTggQzQzLjI5MiwxMTMuMDY1IDQzLjA2MSwxMTMuMTMxIDQyLjg5NiwxMTMuMzYyIEM0Mi40NjcsMTE0LjIyIDQyLjI2OSwxMTUuMjQzIDQyLjI2OSwxMTYuMjMzIEw0Mi4yMDMsMTIwLjIyNiBDNDIuMTk2NCwxMjAuNzgwNCA0Mi4xOTM3NiwxMjEuNDI5ODQgNDIuMTkzMjMyLDEyMi4xMzcwOTYgTDQyLjE5Mzc5MywxMjMuMjM2Njg5IEM0Mi4xOTQ2MzQ1LDEyMy44MDM1MTMgNDIuMTk2MTQ0MiwxMjQuMzkyMzkgNDIuMTk3NTQyNiwxMjQuOTg3NjE1IEw0Mi4xOTk5NTk5LDEyNi4xODE4NzcgQzQyLjIwMjY3LDEyNy45NzIwMDggNDIuMTk5NywxMjkuNzI1MDUgNDIuMTcsMTMxLjAxNyBDNDIuMTM3LDEzMS44NDIgNDIuMDA1LDEzMy4yOTQgNDIuMDA1LDEzNC4wNTMgWiBNNDMuMjkyLDEyMi41NjkgQzQzLjI5MiwxMjEuMzgxIDQzLjM1OCwxMTguOTM5IDQzLjM5MSwxMTcuNjE5IEM0NC41NzksMTE2LjA2OCA0Ni4wMzEsMTE0Ljc0OCA0Ny42NDgsMTEzLjk4OSBDNDguNjcxLDExMy40OTQgNDkuNzkzLDExMy4yMyA1MC45NDgsMTEzLjEzMSBDNTEuODcyLDExMy4wNjUgNTIuNjk3LDExMy4zMjkgNTMuMjI1LDExNC4wNTUgQzUzLjU4OCwxMTQuNTgzIDUzLjYyMSwxMTUuMTc3IDUzLjUyMiwxMTUuNzcxIEM1My4yNTgsMTE3LjA5MSA1Mi4yMzUsMTE4LjQ0NCA1MS4yNzgsMTE5LjMzNSBDNTAuMzg3LDEyMC4xOTMgNDkuMzk3LDEyMC44NTMgNDguMzA4LDEyMS4zODEgQzQ3LjMxOCwxMjEuODQzIDQ2LjI2MiwxMjIuMTczIDQ1LjIwNiwxMjIuNDM3IEw0NC4yMTYsMTIyLjU2OSBDNDMuOTE5LDEyMi42MDIgNDMuNTg5LDEyMi42MDIgNDMuMjkyLDEyMi41NjkgWiBNNTkuMTk4LDEzMy42NTcgQzYxLjI3NywxMzIuMDA3IDYyLjQ2NSwxMjkuNjk3IDYzLjM1NiwxMjcuNDUzIEM2My41MjEsMTI4LjE3OSA2My42ODYsMTI4LjkwNSA2My45MTcsMTI5LjU5OCBDNjQuNDc4LDEzMS4xODIgNjQuOTQsMTMyLjY2NyA2Ni4zOTIsMTMzLjcyMyBDNjYuNjg5LDEzMy43ODkgNjcuMjE3LDEzMy44NTUgNjcuMTg0LDEzMy40MjYgQzY1LjgzMSwxMzIuMTA2IDY1LjUwMSwxMzAuNzUzIDY0Ljk0LDEyOS4wMDQgQzY0LjYxLDEyNy45MTUgNjQuMzQ2LDEyNi44NTkgNjQuMjE0LDEyNS42NzEgQzY0LjExNSwxMjUuMzc0IDY0LjA0OSwxMjUuMTc2IDYzLjkxNywxMjQuOTQ1IEM2My44NTEsMTI0LjM4NCA2My44NTEsMTIzLjY5MSA2My41MjEsMTIzLjIyOSBDNjMuMjI0LDEyMi40MzcgNjIuNjYzLDEyMS43NDQgNjEuODcxLDEyMS4zODEgQzYxLjU0MSwxMjEuMjE2IDYxLjIxMSwxMjEuMjE2IDYwLjg4MSwxMjEuMzE1IEM2MC41ODQsMTIxLjQxNCA2MC4yODcsMTIxLjU3OSA1OS45OSwxMjEuNzQ0IEM1Ny44NDUsMTIzLjU1OSA1Ni40NTksMTI1Ljg2OSA1NS43OTksMTI4LjM3NyBDNTUuNTM1LDEyOS4zMzQgNTUuMzM3LDEzMC42MjEgNTUuNDY5LDEzMS43NDMgQzU1LjYwMSwxMzIuNjM0IDU1Ljk2NCwxMzMuNDI2IDU2Ljc4OSwxMzMuOTIxIEM1Ny41MTUsMTM0LjM4MyA1OC41MzgsMTM0LjE4NSA1OS4xOTgsMTMzLjY1NyBaIE01Ny4wNTMsMTMyLjYzNCBDNTYuNzg5LDEzMi4yNzEgNTYuNjU3LDEzMS44NDIgNTYuNjI0LDEzMS4zNDcgQzU2LjU5MSwxMzAuODUyIDU2LjY1NywxMzAuMzI0IDU2LjY5LDEyOS44MjkgQzU2Ljk4NywxMjguMzQ0IDU3LjQ4MiwxMjYuOTI1IDU4LjIwOCwxMjUuNjA1IEM1OC44MDIsMTI0LjU4MiA1OS43MjYsMTIzLjMyOCA2MC43NDksMTIyLjYzNSBDNjEuMDEzLDEyMi40NyA2MS4yNzcsMTIyLjM3MSA2MS42MDcsMTIyLjQ3IEM2Mi4wMzYsMTIyLjcwMSA2Mi4zNjYsMTIzLjEzIDYyLjYzLDEyMy41OTIgQzYyLjgyOCwxMjMuOTU1IDYzLjA1OSwxMjQuNTQ5IDYzLjE1OCwxMjQuODc5IEM2Mi45MjcsMTI1LjIwOSA2Mi44OTQsMTI1LjYwNSA2Mi44MjgsMTI1Ljk2OCBDNjEuOTcsMTI4LjU0MiA2MC42MTcsMTMxLjA1IDU4LjI3NCwxMzIuODY1IEM1Ny44NzgsMTMyLjk5NyA1Ny4zNSwxMzIuOTMxIDU3LjA1MywxMzIuNjM0IFogTTcyLjEzNCwxNDEuNDQ1IEM3My4xOSwxNDAuODg0IDczLjg1LDE0MC4xMjUgNzQuMjQ2LDEzOS4xNjggQzc0LjYwOSwxMzguMjQ0IDc0Ljc0MSwxMzcuMTg4IDc0LjgwNywxMzYuMDY2IEM3NC44NCwxMzUuMDEgNzQuODQsMTMzLjg4OCA3NC44NCwxMzIuNzk5IEM3NS4wMzgsMTMyLjYwMSA3NS4zMDIsMTMyLjQwMyA3NS41MzMsMTMyLjIwNSBDNzUuNzk3LDEzMS45NzQgNzYuMDYxLDEzMS43NDMgNzYuMjU5LDEzMS40NzkgQzc2LjM5MSwxMzEuMjgxIDc2LjMyNSwxMzAuODg1IDc2LjEyNywxMzAuODg1IEM3NS44OTYsMTMxLjAxNyA3NS42NjUsMTMxLjE0OSA3NS40MzQsMTMxLjMxNCBDNzUuMjM2LDEzMS40MTMgNzUuMDM4LDEzMS41NDUgNzQuODczLDEzMS42MTEgQzc0Ljg3MywxMjkuODYyIDc0Ljk3MiwxMjguMjEyIDc1LjEzNywxMjYuNTI5IEM3NS4wNzEsMTI2LjEzMyA3NS4yNjksMTI1LjcwNCA3NS4xNywxMjUuMzA4IEM3NS4wNzEsMTI1LjA0NCA3NC44MDcsMTI0LjgxMyA3NC41MSwxMjQuNzggQzc0LjM0NSwxMjQuNzggNzQuMTgsMTI0Ljg0NiA3NC4wMTUsMTI0LjkxMiBDNzQuMTgsMTI0LjEyIDc0LjM3OCwxMjMuMTYzIDc0LjAxNSwxMjIuMzcxIEM3My43ODQsMTIxLjkwOSA3My4zMjIsMTIxLjU3OSA3Mi45MjYsMTIxLjM4MSBDNzIuMzY1LDEyMS4yNDkgNzEuNzA1LDEyMS4zMTUgNzEuMjQzLDEyMS42MTIgQzcwLjUxNywxMjIuMTA3IDY5Ljk1NiwxMjIuNzY3IDY5LjQ5NCwxMjMuMzk0IEM2OC42NjksMTI0LjU0OSA2OC4xNzQsMTI1Ljg2OSA2OC4wNDIsMTI3LjI4OCBDNjguMDQyLDEyOC4yMTIgNjguNDM4LDEyOS4xMzYgNjkuMzYyLDEyOS41MzIgQzcwLjIyLDEyOS43MyA3MC45NDYsMTI5LjQ5OSA3MS41NzMsMTI5LjAzNyBDNzIuMTAxLDEyOC42NDEgNzIuNTYzLDEyOC4wOCA3My4wMjUsMTI3LjQ4NiBMNzQuMDgxLDEyNi4wNjcgQzczLjk4MiwxMjguMTQ2IDczLjc1MSwxMzAuMjU4IDczLjc4NCwxMzIuMzM3IEM3Mi4yNjYsMTMzLjUyNSA3MC43MTUsMTM0LjgxMiA2OS42MjYsMTM2LjQ2MiBDNjguOTMzLDEzNy41MTggNjguMTc0LDEzOC43MDYgNjguNDA1LDE0MC4xNTggQzY4LjYzNiwxNDAuODE4IDY5LjE2NCwxNDEuMzc5IDY5LjgyNCwxNDEuNjc2IEM3MC41MTcsMTQxLjk3MyA3MS41MDcsMTQxLjgwOCA3Mi4xMzQsMTQxLjQ0NSBaIE02OS45NTYsMTI4LjU0MiBDNjkuNjkyLDEyOC40NDMgNjkuMzk1LDEyOC4yNDUgNjkuMzI5LDEyNy45ODEgQzY5LjE2NCwxMjcuMTg5IDY5LjI2MywxMjYuNDYzIDY5LjUyNywxMjUuNzcgQzY5Ljg1NywxMjQuODEzIDcwLjc0OCwxMjMuMTk2IDcxLjYzOSwxMjIuNjAyIEM3Mi4wMzUsMTIyLjMzOCA3Mi41NjMsMTIyLjI3MiA3My4wMjUsMTIyLjczNCBDNzMuMjIzLDEyMy4xNjMgNzMuMjg5LDEyMy42MjUgNzMuMTksMTI0LjEyIEM3My4xNTcsMTI0LjQ4MyA3My4wMjUsMTI0Ljg0NiA3Mi44OTMsMTI1LjIwOSBDNzIuNzYxLDEyNS42MDUgNzIuNTk2LDEyNS45NjggNzIuNDY0LDEyNi4zMzEgQzcyLjEzNCwxMjYuOTI1IDcxLjc3MSwxMjcuNTg1IDcxLjI0MywxMjguMDggQzcwLjkxMywxMjguNDEgNzAuNDg0LDEyOC41NzUgNjkuOTU2LDEyOC41NDIgWiBNNjkuOTIzLDE0MC40NTUgQzY5LjU5MywxNDAuMTkxIDY5LjUyNywxMzkuNzk1IDY5LjQ5NCwxMzkuMzY2IEM2OS42MjYsMTM4LjU0MSA3MC4wMjIsMTM3LjgxNSA3MC40ODQsMTM3LjEyMiBDNzEuMzc1LDEzNS44MDIgNzIuNDY0LDEzNC42OCA3My43MTgsMTMzLjYyNCBDNzMuNzUxLDEzNC4zODMgNzMuNzg0LDEzNS4yMDggNzMuNzUxLDEzNiBDNzMuNzE4LDEzNi44MjUgNzMuNTg2LDEzNy42NSA3My4zMjIsMTM4LjM3NiBDNzIuOTkyLDEzOS4zNjYgNzIuMjk5LDE0MC4yMjQgNzEuMTc3LDE0MC43MTkgQzcwLjcxNSwxNDAuODE4IDcwLjI1MywxNDAuNzE5IDY5LjkyMywxNDAuNDU1IFogTTgzLjIyMiwxMzQuMTUyIEM4NC4yNDUsMTM0LjA1MyA4NS4yMzUsMTMzLjU1OCA4Ni4xMjYsMTMyLjg2NSBDODYuNzg2LDEzMi4zMzcgODcuMzgsMTMxLjcxIDg3Ljg0MiwxMzEuMDE3IEM4OC4xNzIsMTMwLjQ4OSA4OC41NjgsMTI5LjY5NyA4OC42MDEsMTI5LjA3IEM4OC42MDEsMTI4LjkzOCA4OC4zMDQsMTI4LjYwOCA4OC4xMzksMTI4Ljc3MyBDODcuNzQzLDEyOS44OTUgODYuOTE4LDEzMC44NTIgODUuOTI4LDEzMS43MSBDODUuNCwxMzIuMTcyIDg0LjcwNywxMzIuNjAxIDgzLjk4MSwxMzIuODY1IEM4My4yMjIsMTMzLjE2MiA4Mi4zOTcsMTMzLjIyOCA4MS42MDUsMTMyLjc2NiBDODAuNzQ3LDEzMi4yNzEgODAuNDUsMTMxLjM0NyA4MC4yODUsMTMwLjQyMyBDODAuODQ2LDEzMC4wNiA4MS40NzMsMTI5LjYzMSA4Mi4xLDEyOS4xNjkgQzgyLjcyNywxMjguNzA3IDgzLjM1NCwxMjguMjEyIDgzLjg4MiwxMjcuNzE3IEM4My45ODEsMTI3LjU4NSA4NC4xNDYsMTI3LjQyIDg0LjI3OCwxMjcuMjg4IEw4NC43NzMsMTI2LjgyNiBDODUuMTM2LDEyNi40OTYgODUuNDk5LDEyNi4xMzMgODUuNzk2LDEyNS44MDMgQzg2LjE1OSwxMjUuMzc0IDg2LjQ1NiwxMjQuODEzIDg2LjMyNCwxMjQuMTIgQzg1Ljg2MiwxMjMuMDY0IDg1LjAwNCwxMjEuODQzIDgzLjg0OSwxMjEuNDQ3IEM4Mi45NTgsMTIxLjE1IDgxLjU3MiwxMjEuNTQ2IDgwLjg0NiwxMjIuMTQgQzgwLjA1NCwxMjMuMDMxIDc5LjUyNiwxMjQuMTg2IDc5LjI5NSwxMjUuMzQxIEM3OC45NjUsMTI2LjgyNiA3OC45NjUsMTI4LjQxIDc5LjA2NCwxMjkuODk1IEM3OC44LDEzMC4wMjcgNzguNDcsMTMwLjI1OCA3OC4yMzksMTMwLjM1NyBDNzguMTA3LDEzMC40MjMgNzcuOTQyLDEzMC40NTYgNzcuNzQ0LDEzMC41MjIgQzc3LjU0NiwxMzAuNTU1IDc3LjM0OCwxMzAuNjIxIDc3LjE1LDEzMC43MiBDNzYuOTg1LDEzMC45ODQgNzcuMTE3LDEzMS4yODEgNzcuMjQ5LDEzMS40MTMgQzc3LjY3OCwxMzEuNjExIDc3Ljk3NSwxMzEuNjExIDc4LjMzOCwxMzEuNTEyIEM3OC41NjksMTMxLjQ0NiA3OC45MzIsMTMxLjI0OCA3OS4xOTYsMTMxLjA1IEM3OS4yOTUsMTMxLjM4IDc5LjM5NCwxMzEuNzEgNzkuNTU5LDEzMi4wMDcgQzc5LjcyNCwxMzIuMzcgNzkuOTg4LDEzMi43MzMgODAuMzg0LDEzMy4xNjIgQzgxLjA3NywxMzMuODg4IDgyLjEzMywxMzQuMjUxIDgzLjIyMiwxMzQuMTUyIFogTTgwLjE4NiwxMjkuMTAzIEM4MC4xMiwxMjcuODQ5IDgwLjE1MywxMjYuNTk1IDgwLjQ4MywxMjUuNDQgQzgwLjcxNCwxMjQuNTQ5IDgxLjA0NCwxMjMuNzU3IDgxLjU3MiwxMjMuMDY0IEM4Mi4xLDEyMi41NjkgODMuMTg5LDEyMi40MzcgODMuODQ5LDEyMi42NjggQzg0LjQ3NiwxMjMuMDk3IDg0LjkzOCwxMjMuNzkgODUuMjM1LDEyNC40NSBDODUuMTM2LDEyNS4xMSA4NC41MDksMTI1LjU3MiA4NC4wOCwxMjYuMDAxIEM4My41MTksMTI2LjU5NSA4Mi44NTksMTI3LjE4OSA4Mi4xMzMsMTI3LjcxNyBDODEuNDczLDEyOC4yMTIgODAuODEzLDEyOC42NzQgODAuMTg2LDEyOS4xMDMgWiBNMTAzLjI4NiwxMzQuMjE4IEMxMDUuMjMzLDEzNC4wODYgMTA3LjAxNSwxMzMuNTI1IDEwOC42NjUsMTMyLjY2NyBDMTEwLjM4MSwxMzEuODA5IDExMS4xNCwxMzEuMzE0IDExMi42OTEsMTMwLjAyNyBDMTEyLjkyMiwxMjkuODI5IDExMy4yMTksMTI5LjU5OCAxMTMuNDgzLDEyOS4zMzQgQzExMy43OCwxMjkuMDM3IDExNC4wNDQsMTI4Ljc0IDExNC4yNDIsMTI4LjM3NyBDMTE0LjMwOCwxMjguMTc5IDExNC4wNDQsMTI4LjAxNCAxMTMuODc5LDEyOC4wOCBDMTEyLjAzMSwxMjkuNTk4IDExMC43NDQsMTMwLjUyMiAxMDguNzY0LDEzMS41MTIgQzEwNy42MDksMTMyLjEwNiAxMDYuMzg4LDEzMi41MzUgMTA1LjEzNCwxMzIuODMyIEMxMDQuMjc2LDEzMy4wNjMgMTAzLjE4NywxMzMuMzYgMTAyLjI5NiwxMzIuOTMxIEMxMDEuNjAzLDEzMi42MDEgMTAxLjIwNywxMzEuOTA4IDEwMS4xNzQsMTMxLjIxNSBDMTAwLjk0MywxMjguMTc5IDEwMS4xNzQsMTIzLjYyNSAxMDEuNjM2LDExOS45MjkgTDEwMi41MjcsMTEyLjY2OSBDMTAyLjQ5NCwxMTIuNDM4IDEwMi4yNjMsMTEyLjEwOCAxMDEuOTY2LDExMi4xMDggQzEwMS42MzYsMTEyLjIwNyAxMDEuNTM3LDExMi41MDQgMTAxLjQzOCwxMTIuNzM1IEMxMDAuMzgyLDExOC43NDEgOTkuNjg5LDEyNi4zMzEgMTAwLjAxOSwxMzEuMjE1IEMxMDAuMDg1LDEzMS45NzQgMTAwLjI4MywxMzIuNjM0IDEwMC43NDUsMTMzLjE5NSBDMTAxLjM3MiwxMzMuOTg3IDEwMi4yOTYsMTM0LjI4NCAxMDMuMjg2LDEzNC4yMTggWiBNMTIyLjE2MiwxMzIuNTY4IEMxMjMuMjE4LDEzMS40MTMgMTIzLjk0NCwxMjkuOTI4IDEyNC4yNDEsMTI4LjM3NyBDMTI0LjUzOCwxMjYuNzYgMTI0LjM3MywxMjUuMTEgMTIzLjY4LDEyMy42MjUgQzEyMy40MTYsMTIzLjAzMSAxMjIuOTU0LDEyMi40MDQgMTIyLjM2LDEyMS45NzUgQzEyMi4wMywxMjEuNzQ0IDEyMS42MzQsMTIxLjU0NiAxMjEuMjA1LDEyMS40OCBDMTIxLjEwNiwxMjEuMjQ5IDEyMC43NzYsMTIxLjE1IDEyMC41NzgsMTIxLjE4MyBDMTE5LjU4OCwxMjEuNDE0IDExOC43MywxMjIuMDc0IDExNy45NzEsMTIyLjggQzExNi42ODQsMTI0LjAyMSAxMTUuNzYsMTI1LjUzOSAxMTUuMjY1LDEyNy4xNTYgQzExNC44MDMsMTI4Ljc0IDExNC43NywxMzAuNDIzIDExNS4zNjQsMTMyLjEwNiBDMTE1LjcyNywxMzMuMDk2IDExNi43NSwxMzMuOTU0IDExNy43NzMsMTM0LjE1MiBDMTE5LjUyMiwxMzQuNDQ5IDEyMS4wMDcsMTMzLjc4OSAxMjIuMTYyLDEzMi41NjggWiBNMTE4LjAwNCwxMzIuOTk3IEMxMTcuMjQ1LDEzMi44MzIgMTE2LjQ4NiwxMzIuMjM4IDExNi4yODgsMTMxLjQ0NiBDMTE1Ljk1OCwxMjkuOTYxIDExNi4wMjQsMTI4LjQ3NiAxMTYuNDUzLDEyNy4xNTYgQzExNy4wMTQsMTI1LjUwNiAxMTcuOTM4LDEyMy45NTUgMTE5LjQ4OSwxMjIuOTY1IEwxMTkuOTUxLDEyMi42NjggQzEyMC4xMTYsMTIyLjU2OSAxMjAuMjgxLDEyMi40NyAxMjAuNDQ2LDEyMi4zMzggQzEyMC43MSwxMjIuMTczIDEyMS4wNCwxMjIuMTczIDEyMS4zNywxMjIuNTAzIEMxMjIuMjYxLDEyMy4xNjMgMTIyLjk1NCwxMjQuMDIxIDEyMy4xMTksMTI1LjA0NCBDMTIzLjMxNywxMjYuMTk5IDEyMy4zMTcsMTI3LjQyIDEyMy4wNTMsMTI4LjU0MiBDMTIyLjc1NiwxMjkuOTYxIDEyMi4wNjMsMTMxLjI0OCAxMjAuOTA4LDEzMi4yMzggQzEyMC4xNDksMTMyLjg5OCAxMTkuMDkzLDEzMy4yNjEgMTE4LjAwNCwxMzIuOTk3IFogTTEyOS40MjIsMTMzLjY1NyBDMTMxLjUwMSwxMzIuMDA3IDEzMi42ODksMTI5LjY5NyAxMzMuNTgsMTI3LjQ1MyBDMTMzLjc0NSwxMjguMTc5IDEzMy45MSwxMjguOTA1IDEzNC4xNDEsMTI5LjU5OCBDMTM0LjcwMiwxMzEuMTgyIDEzNS4xNjQsMTMyLjY2NyAxMzYuNjE2LDEzMy43MjMgQzEzNi45MTMsMTMzLjc4OSAxMzcuNDQxLDEzMy44NTUgMTM3LjQwOCwxMzMuNDI2IEMxMzYuMDU1LDEzMi4xMDYgMTM1LjcyNSwxMzAuNzUzIDEzNS4xNjQsMTI5LjAwNCBDMTM0LjgzNCwxMjcuOTE1IDEzNC41NywxMjYuODU5IDEzNC40MzgsMTI1LjY3MSBDMTM0LjMzOSwxMjUuMzc0IDEzNC4yNzMsMTI1LjE3NiAxMzQuMTQxLDEyNC45NDUgQzEzNC4wNzUsMTI0LjM4NCAxMzQuMDc1LDEyMy42OTEgMTMzLjc0NSwxMjMuMjI5IEMxMzMuNDQ4LDEyMi40MzcgMTMyLjg4NywxMjEuNzQ0IDEzMi4wOTUsMTIxLjM4MSBDMTMxLjc2NSwxMjEuMjE2IDEzMS40MzUsMTIxLjIxNiAxMzEuMTA1LDEyMS4zMTUgQzEzMC44MDgsMTIxLjQxNCAxMzAuNTExLDEyMS41NzkgMTMwLjIxNCwxMjEuNzQ0IEMxMjguMDY5LDEyMy41NTkgMTI2LjY4MywxMjUuODY5IDEyNi4wMjMsMTI4LjM3NyBDMTI1Ljc1OSwxMjkuMzM0IDEyNS41NjEsMTMwLjYyMSAxMjUuNjkzLDEzMS43NDMgQzEyNS44MjUsMTMyLjYzNCAxMjYuMTg4LDEzMy40MjYgMTI3LjAxMywxMzMuOTIxIEMxMjcuNzM5LDEzNC4zODMgMTI4Ljc2MiwxMzQuMTg1IDEyOS40MjIsMTMzLjY1NyBaIE0xMjcuMjc3LDEzMi42MzQgQzEyNy4wMTMsMTMyLjI3MSAxMjYuODgxLDEzMS44NDIgMTI2Ljg0OCwxMzEuMzQ3IEMxMjYuODE1LDEzMC44NTIgMTI2Ljg4MSwxMzAuMzI0IDEyNi45MTQsMTI5LjgyOSBDMTI3LjIxMSwxMjguMzQ0IDEyNy43MDYsMTI2LjkyNSAxMjguNDMyLDEyNS42MDUgQzEyOS4wMjYsMTI0LjU4MiAxMjkuOTUsMTIzLjMyOCAxMzAuOTczLDEyMi42MzUgQzEzMS4yMzcsMTIyLjQ3IDEzMS41MDEsMTIyLjM3MSAxMzEuODMxLDEyMi40NyBDMTMyLjI2LDEyMi43MDEgMTMyLjU5LDEyMy4xMyAxMzIuODU0LDEyMy41OTIgQzEzMy4wNTIsMTIzLjk1NSAxMzMuMjgzLDEyNC41NDkgMTMzLjM4MiwxMjQuODc5IEMxMzMuMTUxLDEyNS4yMDkgMTMzLjExOCwxMjUuNjA1IDEzMy4wNTIsMTI1Ljk2OCBDMTMyLjE5NCwxMjguNTQyIDEzMC44NDEsMTMxLjA1IDEyOC40OTgsMTMyLjg2NSBDMTI4LjEwMiwxMzIuOTk3IDEyNy41NzQsMTMyLjkzMSAxMjcuMjc3LDEzMi42MzQgWiBNMTQyLjc4NywxMzIuNTM1IEMxNDQuMzM4LDEzMC4yOTEgMTQ1LjM2MSwxMjcuODE2IDE0Ni4yMTksMTI1LjI3NSBDMTQ2LjM1MSwxMjYuNTI5IDE0Ni4zMTgsMTI3Ljc4MyAxNDYuNTE2LDEyOC45NzEgQzE0Ni42ODEsMTI5Ljg2MiAxNDYuOTEyLDEzMC42ODcgMTQ3LjUwNiwxMzEuNDEzIEMxNDcuNjM4LDEzMS41NzggMTQ4LjAwMSwxMzEuNzc2IDE0OC4zNjQsMTMxLjU3OCBDMTQ4LjMzMSwxMzEuNDEzIDE0OC4yNjUsMTMxLjI4MSAxNDguMjMyLDEzMS4xODIgQzE0OC4xLDEzMC44NTIgMTQ3LjkwMiwxMzAuNTU1IDE0Ny44MzYsMTMwLjE5MiBDMTQ3LjIwOSwxMjcuMjU1IDE0Ny4yNDIsMTI0LjE4NiAxNDcuMzQxLDEyMS4xNSBDMTQ3LjQwMDQsMTE5LjQyNzQgMTQ3LjYyNjEyLDExNy43NjQyIDE0Ny44NzU2LDExNi4xMDMzNzYgTDE0OC4xMjc3MiwxMTQuNDQxMjMyIEMxNDguMjEwODgsMTEzLjg4NjA0IDE0OC4yOTE0LDExMy4zMjkgMTQ4LjM2NCwxMTIuNzY4IEMxNDguMzk3LDExMi40MzggMTQ3LjgwMywxMTEuODc3IDE0Ny40NCwxMTIuNDM4IEMxNDcuMTQzLDExMy4zOTUgMTQ2Ljk3OCwxMTQuMzg1IDE0Ni44NDYsMTE1LjM3NSBDMTQ2LjcxNCwxMTYuMzY1IDE0Ni42MTUsMTE3LjM4OCAxNDYuNDUsMTE4LjQxMSBDMTQ2LjMxOCwxMTkuNDM0IDE0Ni4zMTgsMTIwLjQ5IDE0Ni4yNTIsMTIxLjUxMyBDMTQ2LjIxOSwxMjIuMjM5IDE0Ni4wODcsMTIyLjk2NSAxNDUuODU2LDEyMy41OTIgQzE0NS43NTcsMTIzLjE5NiAxNDUuNjI1LDEyMi44MzMgMTQ1LjM2MSwxMjIuNTAzIEMxNDQuODY2LDEyMS45MDkgMTQ0LjA3NCwxMjEuNDggMTQzLjI4MiwxMjEuNDQ3IEMxNDEuODk2LDEyMS4zNDggMTQwLjc3NCwxMjIuNDM3IDE0MC4wODEsMTIzLjU1OSBDMTM5LjE1NywxMjUuMDExIDEzOC4zOTgsMTI2Ljc2IDEzOC4xMzQsMTI4LjU3NSBDMTM3LjkwMywxMjkuOTk0IDEzNy45NjksMTMxLjQ0NiAxMzguNDY0LDEzMi44OTggQzEzOC44NiwxMzMuNTU4IDEzOS41MiwxMzQuMDIgMTQwLjI0NiwxMzQuMDg2IEMxNDEuMzAyLDEzNC4xODUgMTQyLjE5MywxMzMuMzkzIDE0Mi43ODcsMTMyLjUzNSBaIE0xNDAuMDgxLDEzMi44OTggQzEzOS42ODUsMTMyLjcgMTM5LjQ1NCwxMzIuMzcgMTM5LjMyMiwxMzEuOTQxIEMxMzkuMTksMTMxLjU3OCAxMzkuMTI0LDEzMS4xMTYgMTM5LjEyNCwxMzAuNjU0IEMxMzkuMTI0LDEzMC4xOTIgMTM5LjE1NywxMjkuNzMgMTM5LjE1NywxMjkuMzAxIEMxMzkuNDg3LDEyNy42MTggMTM5LjkxNiwxMjYuMDAxIDE0MC43NzQsMTI0LjU0OSBDMTQxLjE3LDEyMy45MjIgMTQxLjY2NSwxMjMuMzI4IDE0Mi4yOTIsMTIyLjc2NyBDMTQyLjY1NSwxMjIuNTM2IDE0My4xNSwxMjIuNDM3IDE0My41NzksMTIyLjYwMiBDMTQ0LjAwOCwxMjIuNzM0IDE0NC40MDQsMTIyLjk5OCAxNDQuNjAyLDEyMy4zOTQgQzE0NC44NjYsMTIzLjk1NSAxNDQuNzY3LDEyNC42ODEgMTQ0Ljc2NywxMjUuMjc1IEMxNDQuNzY3LDEyNS40MDcgMTQ0Ljg2NiwxMjUuNjcxIDE0NS4wMzEsMTI1LjczNyBDMTQ0LjIwNiwxMjguMDE0IDE0My4yNDksMTMwLjI1OCAxNDEuNjk4LDEzMi4xNzIgQzE0MS4yMzYsMTMyLjczMyAxNDAuNzA4LDEzMy4wOTYgMTQwLjA4MSwxMzIuODk4IFogTTE1OC4xMzIsMTE1LjI3NiBDMTU4LjE5OCwxMTQuODggMTU3Ljk2NywxMTQuNTgzIDE1Ny43NjksMTE0LjQxOCBDMTU2LjMxNywxMTMuMDk4IDE1NC41NjgsMTEyLjQwNSAxNTIuNTg4LDExMi4zNzIgQzE1Mi4yNTgsMTEyLjYzNiAxNTIuMjkxLDExMy4yOTYgMTUyLjYyMSwxMTMuMzI5IEMxNTQuMDczLDExMy42MjYgMTU1LjQyNiwxMTMuOTg5IDE1Ni43NDYsMTE0LjU4MyBMMTU3LjM3MywxMTQuOTQ2IEMxNTcuNTcxLDExNS4wNDUgMTU3LjgwMiwxMTUuMTc3IDE1OC4xMzIsMTE1LjI3NiBaIE0xNTUuOTIxLDEzNC4wODYgQzE1Ni40NDksMTMzLjk1NCAxNTYuOTQ0LDEzMy42MjQgMTU3LjMwNywxMzMuMjI4IEMxNTcuODM1LDEzMi42MzQgMTU4LjI2NCwxMzEuOTA4IDE1OC4zOTYsMTMxLjI0OCBMMTU4LjUyOCwxMzAuNjU0IEMxNTguNTYxLDEzMC40NTYgMTU4LjU2MSwxMzAuMjI1IDE1OC4yOTcsMTI5Ljk5NCBDMTU4LjAzMywxMzAuMjI1IDE1Ny45MDEsMTMwLjQ4OSAxNTcuODAyLDEzMC43MiBDMTU3LjcwMywxMzAuOTUxIDE1Ny42MzcsMTMxLjIxNSAxNTcuNTA1LDEzMS40NDYgQzE1Ny4yNDEsMTMxLjgwOSAxNTYuOTc3LDEzMi4yMzggMTU2LjYxNCwxMzIuNTM1IEMxNTYuMjg0LDEzMi43OTkgMTU1Ljk1NCwxMzMuMDMgMTU1LjU1OCwxMzIuOTk3IEMxNTUuMzYsMTMyLjk5NyAxNTQuOTY0LDEzMi42MzQgMTU0LjkzMSwxMzIuNDM2IEMxNTQuMjcxLDEzMC4xOTIgMTU0LjYzNCwxMjUuNzM3IDE1NS4wNjMsMTIyLjQwNCBDMTU1LjA5NiwxMjIuMTQgMTU1LjEyOSwxMjEuODEgMTU0Ljg5OCwxMjEuNTQ2IEMxNTQuNjY3LDEyMS40MTQgMTU0LjE3MiwxMjEuMjgyIDE1NC4wMDcsMTIxLjU0NiBDMTUzLjE4MiwxMjIuOCAxNTIuNTU1LDEyNC4wODcgMTUxLjgyOSwxMjUuMzc0IEMxNTEuMTAzLDEyNi42NjEgMTUwLjMxMSwxMjcuOTE1IDE0OS4yODgsMTI5LjA3IEMxNDkuMTg5LDEyOS4yMDIgMTQ5LjIyMiwxMjkuMzM0IDE0OS4zMjEsMTI5LjQgQzE0OS40MiwxMjkuNDY2IDE0OS41NTIsMTI5LjQ5OSAxNDkuNjUxLDEyOS40OTkgQzE0OS44ODIsMTI5LjQ2NiAxNTAuMTQ2LDEyOS4zMDEgMTUwLjMxMSwxMjkuMTY5IEMxNTEuMTAzLDEyOC4zMTEgMTUxLjczLDEyNy40MiAxNTIuMjkxLDEyNi40OTYgQzE1Mi43ODYsMTI1LjY3MSAxNTMuMjQ4LDEyNC44NzkgMTUzLjcxLDEyNC4wMjEgQzE1My40NzksMTI2LjcyNyAxNTMuMzQ3LDEyOS40MzMgMTUzLjcxLDEzMi4xMzkgQzE1My44MDksMTMyLjc5OSAxNTQuMDA3LDEzMy4zNiAxNTQuNTM1LDEzMy44MjIgQzE1NC44MzIsMTM0LjA4NiAxNTUuNTI1LDEzNC4xMTkgMTU1LjkyMSwxMzQuMDg2IFogTTE2OS4wMjIsMTM0LjA1MyBDMTcwLjM3NSwxMzIuODk4IDE3MS4zMzIsMTMxLjUxMiAxNzIuMDI1LDEyOS45OTQgQzE3Mi4wNTgsMTI5Ljg2MiAxNzEuODI3LDEyOS41NjUgMTcxLjY2MiwxMjkuNzMgTDE3MC40NDEsMTMxLjMxNCBDMTcwLjA0NSwxMzEuNzQzIDE2OS42NDksMTMyLjE3MiAxNjkuMTg3LDEzMi41MDIgQzE2OS4yODYsMTMxLjE4MiAxNjkuMzUyLDEyOS44MjkgMTY5LjMxOSwxMjguNTA5IEMxNjkuMjg2LDEyNy4xMjMgMTY5LjM4NSwxMjUuNjM4IDE2OS4xODcsMTI0LjI1MiBDMTY5LjA1NSwxMjMuMzk0IDE2OC44MjQsMTIyLjYwMiAxNjguMzk1LDEyMS44NzYgQzE2OC4wMzIsMTIxLjU3OSAxNjcuNjY5LDEyMS4zODEgMTY3LjIwNywxMjEuMzQ4IEMxNjYuNzEyLDEyMS4yODIgMTY2LjM0OSwxMjEuNTQ2IDE2Ni4wMTksMTIxLjc0NCBDMTY0LjkzLDEyMi43NjcgMTY0LjEzOCwxMjMuOTU1IDE2My40NDUsMTI1LjE3NiBDMTYyLjc1MiwxMjYuNDMgMTYyLjE5MSwxMjcuODE2IDE2MS42OTYsMTI5LjE2OSBDMTYxLjQ2NSwxMjYuODU5IDE2MS4zMzMsMTI0LjUxNiAxNjEuMDM2LDEyMi4yMDYgTDE2MC44NzEsMTIxLjg3NiBDMTYwLjcwNiwxMjEuNTEzIDE2MC40NzUsMTIxLjE4MyAxNjAuMDQ2LDEyMS4xMTcgQzE1OS43NDksMTIxLjE4MyAxNTkuNjgzLDEyMS41NDYgMTU5LjcxNiwxMjEuNzExIEMxNjAuMDc5LDEyMi41NjkgMTYwLjE0NSwxMjMuNTI2IDE2MC4yMTEsMTI0LjQ1IEMxNjAuMzEsMTI2LjEzMyAxNjAuNjA3LDEyOS4wMDQgMTYwLjY3MywxMzEuMTgyIEMxNjAuNjczLDEzMS44MDkgMTYwLjcwNiwxMzIuOTY0IDE2MC43MDYsMTMzLjU1OCBDMTYwLjgzOCwxMzQuMDIgMTYxLjQ5OCwxMzMuOTU0IDE2MS42MywxMzMuNTkxIEMxNjEuODYxLDEzMi4wMDcgMTYyLjE5MSwxMzAuNDIzIDE2Mi43MTksMTI4LjkzOCBDMTYzLjM0NiwxMjcuMDkgMTY0LjE3MSwxMjUuMzQxIDE2NS4zOTIsMTIzLjc5IEMxNjUuNzIyLDEyMy4zOTQgMTY2LjA4NSwxMjIuODMzIDE2Ni41MTQsMTIyLjU2OSBDMTY2Ljg3NywxMjIuMzcxIDE2Ny4yNCwxMjIuMzM4IDE2Ny41MzcsMTIyLjYzNSBDMTY3LjY2OSwxMjIuNzM0IDE2Ny43MzUsMTIyLjg5OSAxNjcuODAxLDEyMy4wNjQgQzE2Ny44NjcsMTIzLjIyOSAxNjcuOSwxMjMuMzk0IDE2Ny45MzMsMTIzLjU5MiBDMTY4LjIzLDEyNi40NjMgMTY4LjI5NiwxMjkuNTY1IDE2OC4wOTgsMTMyLjQzNiBDMTY4LjEzMSwxMzIuNTM1IDE2OC4xMzEsMTMyLjYzNCAxNjguMTMxLDEzMi43MzMgQzE2OC4wOTgsMTMzLjAzIDE2OC4wMzIsMTMzLjM5MyAxNjguMDk4LDEzMy42OSBDMTY4LjE5NywxMzMuOTg3IDE2OC42NTksMTM0LjM1IDE2OS4wMjIsMTM0LjA1MyBaIE0xNzcuMDA4LDE0MS40NDUgQzE3OC4wNjQsMTQwLjg4NCAxNzguNzI0LDE0MC4xMjUgMTc5LjEyLDEzOS4xNjggQzE3OS40ODMsMTM4LjI0NCAxNzkuNjE1LDEzNy4xODggMTc5LjY4MSwxMzYuMDY2IEMxNzkuNzE0LDEzNS4wMSAxNzkuNzE0LDEzMy44ODggMTc5LjcxNCwxMzIuNzk5IEMxNzkuOTEyLDEzMi42MDEgMTgwLjE3NiwxMzIuNDAzIDE4MC40MDcsMTMyLjIwNSBDMTgwLjY3MSwxMzEuOTc0IDE4MC45MzUsMTMxLjc0MyAxODEuMTMzLDEzMS40NzkgQzE4MS4yNjUsMTMxLjI4MSAxODEuMTk5LDEzMC44ODUgMTgxLjAwMSwxMzAuODg1IEMxODAuNzcsMTMxLjAxNyAxODAuNTM5LDEzMS4xNDkgMTgwLjMwOCwxMzEuMzE0IEMxODAuMTEsMTMxLjQxMyAxNzkuOTEyLDEzMS41NDUgMTc5Ljc0NywxMzEuNjExIEMxNzkuNzQ3LDEyOS44NjIgMTc5Ljg0NiwxMjguMjEyIDE4MC4wMTEsMTI2LjUyOSBDMTc5Ljk0NSwxMjYuMTMzIDE4MC4xNDMsMTI1LjcwNCAxODAuMDQ0LDEyNS4zMDggQzE3OS45NDUsMTI1LjA0NCAxNzkuNjgxLDEyNC44MTMgMTc5LjM4NCwxMjQuNzggQzE3OS4yMTksMTI0Ljc4IDE3OS4wNTQsMTI0Ljg0NiAxNzguODg5LDEyNC45MTIgQzE3OS4wNTQsMTI0LjEyIDE3OS4yNTIsMTIzLjE2MyAxNzguODg5LDEyMi4zNzEgQzE3OC42NTgsMTIxLjkwOSAxNzguMTk2LDEyMS41NzkgMTc3LjgsMTIxLjM4MSBDMTc3LjIzOSwxMjEuMjQ5IDE3Ni41NzksMTIxLjMxNSAxNzYuMTE3LDEyMS42MTIgQzE3NS4zOTEsMTIyLjEwNyAxNzQuODMsMTIyLjc2NyAxNzQuMzY4LDEyMy4zOTQgQzE3My41NDMsMTI0LjU0OSAxNzMuMDQ4LDEyNS44NjkgMTcyLjkxNiwxMjcuMjg4IEMxNzIuOTE2LDEyOC4yMTIgMTczLjMxMiwxMjkuMTM2IDE3NC4yMzYsMTI5LjUzMiBDMTc1LjA5NCwxMjkuNzMgMTc1LjgyLDEyOS40OTkgMTc2LjQ0NywxMjkuMDM3IEMxNzYuOTc1LDEyOC42NDEgMTc3LjQzNywxMjguMDggMTc3Ljg5OSwxMjcuNDg2IEwxNzguOTU1LDEyNi4wNjcgQzE3OC44NTYsMTI4LjE0NiAxNzguNjI1LDEzMC4yNTggMTc4LjY1OCwxMzIuMzM3IEMxNzcuMTQsMTMzLjUyNSAxNzUuNTg5LDEzNC44MTIgMTc0LjUsMTM2LjQ2MiBDMTczLjgwNywxMzcuNTE4IDE3My4wNDgsMTM4LjcwNiAxNzMuMjc5LDE0MC4xNTggQzE3My41MSwxNDAuODE4IDE3NC4wMzgsMTQxLjM3OSAxNzQuNjk4LDE0MS42NzYgQzE3NS4zOTEsMTQxLjk3MyAxNzYuMzgxLDE0MS44MDggMTc3LjAwOCwxNDEuNDQ1IFogTTE3NC44MywxMjguNTQyIEMxNzQuNTY2LDEyOC40NDMgMTc0LjI2OSwxMjguMjQ1IDE3NC4yMDMsMTI3Ljk4MSBDMTc0LjAzOCwxMjcuMTg5IDE3NC4xMzcsMTI2LjQ2MyAxNzQuNDAxLDEyNS43NyBDMTc0LjczMSwxMjQuODEzIDE3NS42MjIsMTIzLjE5NiAxNzYuNTEzLDEyMi42MDIgQzE3Ni45MDksMTIyLjMzOCAxNzcuNDM3LDEyMi4yNzIgMTc3Ljg5OSwxMjIuNzM0IEMxNzguMDk3LDEyMy4xNjMgMTc4LjE2MywxMjMuNjI1IDE3OC4wNjQsMTI0LjEyIEMxNzguMDMxLDEyNC40ODMgMTc3Ljg5OSwxMjQuODQ2IDE3Ny43NjcsMTI1LjIwOSBDMTc3LjYzNSwxMjUuNjA1IDE3Ny40NywxMjUuOTY4IDE3Ny4zMzgsMTI2LjMzMSBDMTc3LjAwOCwxMjYuOTI1IDE3Ni42NDUsMTI3LjU4NSAxNzYuMTE3LDEyOC4wOCBDMTc1Ljc4NywxMjguNDEgMTc1LjM1OCwxMjguNTc1IDE3NC44MywxMjguNTQyIFogTTE3NC43OTcsMTQwLjQ1NSBDMTc0LjQ2NywxNDAuMTkxIDE3NC40MDEsMTM5Ljc5NSAxNzQuMzY4LDEzOS4zNjYgQzE3NC41LDEzOC41NDEgMTc0Ljg5NiwxMzcuODE1IDE3NS4zNTgsMTM3LjEyMiBDMTc2LjI0OSwxMzUuODAyIDE3Ny4zMzgsMTM0LjY4IDE3OC41OTIsMTMzLjYyNCBDMTc4LjYyNSwxMzQuMzgzIDE3OC42NTgsMTM1LjIwOCAxNzguNjI1LDEzNiBDMTc4LjU5MiwxMzYuODI1IDE3OC40NiwxMzcuNjUgMTc4LjE5NiwxMzguMzc2IEMxNzcuODY2LDEzOS4zNjYgMTc3LjE3MywxNDAuMjI0IDE3Ni4wNTEsMTQwLjcxOSBDMTc1LjU4OSwxNDAuODE4IDE3NS4xMjcsMTQwLjcxOSAxNzQuNzk3LDE0MC40NTUgWiBNMTg3LjMwNCwxMzMuNzg5IEMxODcuNjM0LDEzMy4xMjkgMTg3LjkzMSwxMzIuMTM5IDE4Ny43LDEzMS4zMTQgQzE4Ny42MzQsMTMxLjAxNyAxODcuNDY5LDEzMC41NTUgMTg3LjA0LDEzMC40NTYgQzE4Ni42MTEsMTMwLjM5IDE4Ni4xODIsMTMwLjYyMSAxODUuOTE4LDEzMC45MTggQzE4NS4zMjQsMTMxLjYxMSAxODUuMTkyLDEzMi43IDE4NS40ODksMTMzLjU5MSBDMTg1LjU4OCwxMzMuOTIxIDE4NS44ODUsMTM0LjExOSAxODYuMTQ5LDEzNC4yNTEgQzE4Ni41NzgsMTM0LjM1IDE4Ny4wNzMsMTM0LjE1MiAxODcuMzA0LDEzMy43ODkgWiBNMTk2LjQ3OCwxMzMuNzg5IEMxOTYuODA4LDEzMy4xMjkgMTk3LjEwNSwxMzIuMTM5IDE5Ni44NzQsMTMxLjMxNCBDMTk2LjgwOCwxMzEuMDE3IDE5Ni42NDMsMTMwLjU1NSAxOTYuMjE0LDEzMC40NTYgQzE5NS43ODUsMTMwLjM5IDE5NS4zNTYsMTMwLjYyMSAxOTUuMDkyLDEzMC45MTggQzE5NC40OTgsMTMxLjYxMSAxOTQuMzY2LDEzMi43IDE5NC42NjMsMTMzLjU5MSBDMTk0Ljc2MiwxMzMuOTIxIDE5NS4wNTksMTM0LjExOSAxOTUuMzIzLDEzNC4yNTEgQzE5NS43NTIsMTM0LjM1IDE5Ni4yNDcsMTM0LjE1MiAxOTYuNDc4LDEzMy43ODkgWiBNMjA1LjY1MiwxMzMuNzg5IEMyMDUuOTgyLDEzMy4xMjkgMjA2LjI3OSwxMzIuMTM5IDIwNi4wNDgsMTMxLjMxNCBDMjA1Ljk4MiwxMzEuMDE3IDIwNS44MTcsMTMwLjU1NSAyMDUuMzg4LDEzMC40NTYgQzIwNC45NTksMTMwLjM5IDIwNC41MywxMzAuNjIxIDIwNC4yNjYsMTMwLjkxOCBDMjAzLjY3MiwxMzEuNjExIDIwMy41NCwxMzIuNyAyMDMuODM3LDEzMy41OTEgQzIwMy45MzYsMTMzLjkyMSAyMDQuMjMzLDEzNC4xMTkgMjA0LjQ5NywxMzQuMjUxIEMyMDQuOTI2LDEzNC4zNSAyMDUuNDIxLDEzNC4xNTIgMjA1LjY1MiwxMzMuNzg5IFoiIGlkPSJQYWdlTG9hZGluZy4uLiIgZmlsbD0iIzk5OTk5OSIgZmlsbC1ydWxlPSJub256ZXJvIj48L3BhdGg+CiAgICA8L2c+Cjwvc3ZnPg==`
