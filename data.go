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
	m_cryptonyms                     = make(map[string]string)                       // map[Cryptonym]Definition
	m_words                          = make(map[string]map[string]struct{})          // map[language]map[word]{}
	m_words_english_gematria_english = make(map[string]uint)
	m_words_english_gematria_jewish  = make(map[string]uint)
	m_words_english_gematria_simple  = make(map[string]uint)
	m_gematria_english               = make(map[uint]map[string]struct{})            // english words gematria english values
	m_gematria_jewish                = make(map[uint]map[string]struct{})            // english words gematria jewish values
	m_gematria_simple                = make(map[uint]map[string]struct{})            // english words gematria simple values
	m_collections                    = make(map[string]Collection)
	mu_collections                   = sync.RWMutex{}
	m_collection_documents           = make(map[string]map[string]Document)          // map[CollectionName][DocumentIdentifier]Document{}
	mu_collection_documents          = sync.RWMutex{}
	m_document_pages                 = make(map[string]map[uint]Page)                // map[DocumentIdentifier][PageNumber]Page{}
	mu_document_pages                = sync.RWMutex{}
	m_page_words                     = make(map[string]map[string]struct{})          // map[PageIdentifier]map[word]struct{}
	mu_page_words                    = sync.RWMutex{}
	m_page_gematria_english          = make(map[uint]map[string]map[string]struct{}) // map[GemScore.English]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_english         = sync.RWMutex{}
	m_page_gematria_jewish           = make(map[uint]map[string]map[string]struct{}) // map[GemScore.Jewish]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_jewish          = sync.RWMutex{}
	m_page_gematria_simple           = make(map[uint]map[string]map[string]struct{}) // map[GemScore.Simple]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_simple          = sync.RWMutex{}

	m_location_cities     []*Location
	mu_location_cities    = sync.RWMutex{}
	m_location_countries  []*Location
	mu_location_countries = sync.RWMutex{}
	m_location_states     []*Location
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
	wg_active_tasks = cwg.CountableWaitGroup{}

	// Command Line Flags
	flag_s_database         = config.NewString("database", "", "apario-contribution rendered database directory path")
	flag_i_sem_limiter      = config.NewInt("limit", channel_buffer_size, "general purpose semaphore limiter")
	flag_i_sem_directories  = config.NewInt("directories-limiter", channel_buffer_size, "concurrent directories to process out of the database (example: 369)")
	flag_i_sem_pages        = config.NewInt("pages-limiter", channel_buffer_size, "concurrent pages to process out of the database")
	flag_i_directory_buffer = config.NewInt("directory-buffer", channel_buffer_size, "buffered channel size for pending directories from the database (3x --directories, example: 1107)")
	flag_i_buffer           = config.NewInt("buffer", reader_buffer_bytes, "Memory allocation for CSV buffer (min 168 * 1024 = 168KB)")
	flag_g_log_file         = config.NewString("log", filepath.Join(".", "logs", fmt.Sprintf("badbitchreads-%04d-%02d-%02d-%02d-%02d-%02d.log", startedAt.Year(), startedAt.Month(), startedAt.Day(), startedAt.Hour(), startedAt.Minute(), startedAt.Second())), "File to save logs to. Default is logs/engine-YYYY-MM-DD-HH-MM-SS.log")

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
