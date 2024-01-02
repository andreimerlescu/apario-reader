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
	`math`
	`math/big`
	`regexp`
	`sync`
	"sync/atomic"
	"time"

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

	// Integers
	channel_buffer_size    int = 1          // Buffered Channel's Size
	reader_buffer_bytes    int = 128 * 1024 // 128KB default buffer for reading CSV, XLSX, and PSV files into memory
	jpeg_compression_ratio     = 90         // Progressive JPEG Quality (valid options are 1-100)

	// Colors
	color_background = color.RGBA{R: 40, G: 40, B: 86, A: 255}    // navy blue
	color_text       = color.RGBA{R: 250, G: 226, B: 203, A: 255} // sky yellow

	// Strings

	// Maps
	m_cryptonyms  = make(map[string]string) // map[Cryptonym]Definition
	mu_cryptonyms = sync.RWMutex{}
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

const (
	kilobyte = 1024
	megabyte = 1024 * kilobyte
	gigabyte = 1024 * megabyte
	terabyte = 1024 * gigabyte
	petabyte = 1024 * terabyte
)

func f_s_human_bytes(bytes int64, decimals int) string {
	format := fmt.Sprintf("%%.%df %%s", decimals)

	var result float64
	var suffix string

	switch {
	case bytes < kilobyte:
		return fmt.Sprintf(format, float64(bytes), "B")
	case bytes < megabyte:
		result = float64(bytes) / kilobyte
		suffix = "KB"
	case bytes < gigabyte:
		result = float64(bytes) / megabyte
		suffix = "MB"
	case bytes < terabyte:
		result = float64(bytes) / gigabyte
		suffix = "GB"
	case bytes < petabyte:
		result = float64(bytes) / terabyte
		suffix = "TB"
	default:
		result = float64(bytes) / petabyte
		suffix = "PB"
	}

	result = math.Round(result*math.Pow(10, float64(decimals))) / math.Pow(10, float64(decimals))

	return fmt.Sprintf(format, result, suffix)
}

func f_s_get_page_identifier_from_document_identifier_default_identifier_and_current_page_number(document_identifier string, current_page_identifier string, page_number int) string {
	var page_identifier string
	var page_data map[uint]string
	var is_found bool
	mu_document_pgno_page_identifier.RLock()
	page_data, is_found = m_document_pgno_page_identifier[document_identifier]
	mu_document_pgno_page_identifier.RUnlock()
	if !is_found {
		return current_page_identifier
	}
	var has_page bool
	page_identifier, has_page = page_data[uint(page_number)]
	if !has_page {
		return current_page_identifier
	}
	return page_identifier
}
