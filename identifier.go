package main

import (
	`context`
	`crypto/rand`
	`errors`
	`fmt`
	`log`
	`math`
	`math/big`
	`os`
	`path/filepath`
	`strconv`
	`strings`
	`time`
	`unicode`
)

var sliceRanges = []int{1, 3, 6, 10, 15, 21, 28, 36, 45, 64}

func checksum_to_path(checksum string) (string, error) {
	if len(checksum) != 64 {
		return checksum, fmt.Errorf("invalid checksum length. must be %d bytes", len(checksum))
	}
	var paths []string

	for i := 0; i < len(sliceRanges); i++ {
		r := sliceRanges[i]
		if r > len(checksum) {
			break // Avoid slicing beyond the length of the checksum
		}
		paths = append(paths, checksum[:r])
	}

	return filepath.Join(paths...), nil
}

func identifier_to_path(identifier string) string {
	var paths []string
	var depth int = 1
	for {
		r := fibonacci(depth)
		if r > len(identifier) {
			break
		}
		paths = append(paths, identifier[:r])
		depth++
	}
	return filepath.Join(paths...)
}

func NewIdentifier(database_prefix_path string, length int, attempts int, timeout int) (*TSIdentifier, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()

	ticker := time.NewTicker(33 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			identifier, identifier_err := GenerateIdentifier(database_prefix_path, length, attempts)
			if identifier_err != nil {
				log.Printf("failed to acquire new identifier with err: %v", identifier_err)
			}
			if len(identifier.String()) > 6 {
				return identifier, nil
			}
		}
	}
}

func GenerateIdentifier(database_prefix_path string, length int, attempts int) (*TSIdentifier, error) {
	sem_identifier_generator_concurrency_factor.Acquire()
	defer sem_identifier_generator_concurrency_factor.Release()
	attempted_identifier, attempt_err := NewToken(length, attempts)
	if attempt_err != nil {
		return nil, attempt_err
	}
	identifier := attempted_identifier.String()
	identifier_path := identifier_to_path(identifier)
	path := filepath.Join(database_prefix_path, identifier_path)
	_, info_err := os.Stat(path)
	if errors.Is(info_err, os.ErrNotExist) {
		// directory doesn't exist, so we can create it; this is success
		err := os.MkdirAll(path, 0700)
		if err != nil {
			return nil, err
		}
	} else if errors.Is(info_err, os.ErrPermission) {
		return nil, errors.New(fmt.Sprintf("permission denied on path %v due to err %v", path, info_err))
	} else if info_err != nil {
		log.Printf("error running os.Stat on %v due to err: %v", path, info_err)
		attempts += 1
		if attempts <= 17 {
			return GenerateIdentifier(database_prefix_path, length, attempts)
		}
		return nil, errors.New("failed to acquire new unique identifier within allotted attempt window of opportunity")
	} else if info_err == nil {
		// directory exists
		attempts += 1
		if attempts <= 17 {
			return GenerateIdentifier(database_prefix_path, length, attempts)
		}
		return nil, errors.New("directory exists error 17 times... this is rare! wow")
	}
	return nil, errors.New("unable to secure identifier for use")
}

// NewToken this is attempts squared with a length of the token
func NewToken(length int, attempts int) (*TSIdentifier, error) {
	for {
		token := make([]byte, length)
		for i := range token {
			max := big.NewInt(int64(len(c_identifier_charset)))
			randIndex, err := rand.Int(rand.Reader, max)
			if err != nil {
				log.Printf("failed to generate random number: %v", err)
				continue
			}
			token[i] = c_identifier_charset[randIndex.Int64()]
		}

		id := fmt.Sprintf("%4d%v", time.Now().UTC().Year(), string(token))

		identifier, identifier_err := ParseIdentifier(id)
		if identifier_err != nil {
			attempts += 1
			if attempts <= 17 {
				return NewToken(length, attempts)
			}
			return nil, errors.New("failed to generate acceptable token after 17 attempts")
		}
		return identifier, nil
	}
}

type TSIdentifier struct {
	Year  int8        `json:"y"`
	Shard Base3Number `json:"s"`
	Code  string      `json:"c"`
	Path  string      `json:"p"`
}

// Base3Number represents a number in base 3.
type Base3Number struct {
	Value string `json:"v"`
	Int   int    `json:"i"`
}

func (b *Base3Number) String() string {
	if b.Int == 0 {
		_, err := b.ToInt()
		if err != nil {
			log.Printf("failed to convert base3number toint() for %v", b.Value)
		}
	}
	return fmt.Sprintf("%d", b.Int)
}

// ToInt converts a base 3 number to an integer.
func (b *Base3Number) ToInt() (int, error) {
	if b.Int > 0 {
		return b.Int, nil
	}
	var num int
	for i, digit := range b.Value {
		val := int(digit - '0')
		if val < 0 || val > 2 {
			return 0, fmt.Errorf("invalid digit '%c' in base 3 number", digit)
		}
		num += val * int(math.Pow(3, float64(len(b.Value)-i-1)))
	}
	b.Int = num
	return num, nil
}

// ToHex converts a base 3 number to a hexadecimal string.
func (b *Base3Number) ToHex() (string, error) {
	intValue, err := b.ToInt()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(int64(intValue), 16), nil
}

func (tsi *TSIdentifier) String() string {
	return fmt.Sprintf("%04d%s%v", tsi.Year, tsi.Shard.String(), tsi.Code)
}

func ParseIdentifier(identifier string) (*TSIdentifier, error) {
	path := identifier_to_path(identifier)
	parts := strings.Split(path, string(os.PathSeparator))
	ts_identifier := &TSIdentifier{
		Year:  0,
		Shard: Base3Number{},
		Code:  "",
		Path:  "",
	}
	var year_string string = ""
	var shard_string string = ""
	var code_string string = ""
	var shard int = 0
	var fib_depth int = 1
	for i := 1; i < len(parts); i++ {
		if i > len(parts) {
			break
		}
		r := parts[i]
		if len(r) == 0 {
			break
		}
		fib := fibonacci(fib_depth)
		this_fib_depth := fib_depth
		if len(r) == fib {
			fib_depth += 1
			if this_fib_depth <= 2 {
				year_string = year_string + r
				continue
			} else if this_fib_depth == 3 {
				first_rune := []rune(r)[0]
				if unicode.IsDigit(first_rune) {
					year_string = year_string + string(first_rune)
				}
				second_rune := []rune(r)[1]
				third_rune := []rune(r)[2]
				if unicode.IsDigit(second_rune) && unicode.IsDigit(third_rune) {
					shard_string = fmt.Sprintf("%s%s", string(second_rune), string(third_rune))
				}
				break
			} else {
				code_string = code_string + r
			}
		}
	}
	year, int_err := strconv.Atoi(year_string)
	if int_err != nil {
		return nil, int_err
	}

	// in order for the directory to be considered a year, it must be +/- 17 years from the current date. this value should be configurable
	years := time.Duration(*flag_i_identifier_year_offset)
	if year > time.Now().UTC().AddDate(int(-1*years), 0, 0).Year() && year < time.Now().UTC().Add(years*13*28*24*time.Hour).Year() {
		ts_identifier.Year = int8(year)
	}

	base3_shard := Base3Number{shard_string, 0}
	var shard_err error
	shard, shard_err = base3_shard.ToInt()
	if shard_err != nil {
		return nil, shard_err
	}
	base3_shard.Int = shard
	ts_identifier.Shard = base3_shard
	ts_identifier.Code = code_string
	ts_identifier.Path = path
	return ts_identifier, nil
}

func verify_identifier_path(identifier_path string) error {
	identifier := strings.ReplaceAll(identifier_path, string(os.PathSeparator), ``)
	ts_identifier, identifier_err := ParseIdentifier(identifier)
	if identifier_err != nil {
		return identifier_err
	}
	if ts_identifier.Path != identifier_path {
		return fmt.Errorf("vailed to verify identifier path")
	}
	//_, stat_err := os.Stat(filepath.Join(*flag_s_database))
	return nil
}
