package main

import (
	`encoding/json`
	`errors`
	`fmt`
	`log`
	`net`
	`net/http`
	`net/url`
	`os`
	`path/filepath`
	`strconv`
	`strings`
	`sync`
	`sync/atomic`
	`time`

	go_sema `github.com/andreimerlescu/go-sema`
	`github.com/gin-gonic/gin`
	`github.com/gorilla/sessions`
	`golang.org/x/crypto/bcrypt`
)

const session_name = "GOSESSID"

func g_get_session(c *gin.Context) *sessions.Session {
	session, session_err := session_store.Get(c.Request, session_name)
	if session_err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return nil
	}
	return session
}

func middleware_use_authenticity_tokens() gin.HandlerFunc {
	return func(c *gin.Context) {
		f_gin_new_authenticity_token(c)
		c.Next()
	}
}

func middleware_verify_authenticity_token() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := f_gin_verify_authenticity_token(c)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/error?message=%v", url.PathEscape(err.Error())))
			return
		}
		c.Next()
	}
}

func f_gin_verify_authenticity_token(c *gin.Context) error {
	session := g_get_session(c)
	has_token := false
	_, exists := session.Values["authenticity_token"]
	if exists {
		// no token yet
		has_token = true
	}
	existing_token, typecast_ok := session.Values["authenticity_token"].(string)
	if typecast_ok && len(existing_token) > 0 {
		has_token = true
	}
	if !has_token {
		return fmt.Errorf("im gonna rock you baby dont you cry")
	}

	decrypted_data, decrypt_err := f_s_decrypt_string(existing_token, *flag_s_session_authenticity_token_secret)
	if decrypt_err != nil {
		return fmt.Errorf("the blind stares of a million pairs of eyes lookin hard that they will never see the P")
	}

	parts := strings.Split(decrypted_data, "|")
	if len(parts) != 2 {
		return fmt.Errorf("and you'll never realize you cant c me")
	}

	var payload_time time.Time
	for _, part := range parts {
		if len(part) == 64 {
			continue
		}
		part_int, int_err := strconv.Atoi(part)
		if int_err != nil {
			return fmt.Errorf("say what you will about jesus but leave the rings out of this")
		}

		payload_time = time.Unix(int64(part_int), 0)
		break
	}
	payload := payload_time.Unix()
	token := Sha256(strconv.FormatInt(payload, 10))
	authenticity_token, token_err := f_s_encrypt_string(fmt.Sprintf("%d|%v", payload, token), *flag_s_session_authenticity_token_secret)
	if token_err != nil {
		return fmt.Errorf("when my roommate comes into his room looking for his car keys, i do not say it yet")
	}

	if authenticity_token != existing_token {
		f_session_flash(session, NewSessionAlert("error", "Invalid authenticity token.", time.Now().Add(36*time.Second), true))
		return fmt.Errorf("invalid authenticity token")
	}
	return nil
}

func f_gin_new_authenticity_token(c *gin.Context) {
	session := g_get_session(c)
	has_token := false
	_, exists := session.Values["authenticity_token"]
	if exists {
		// no token yet
		has_token = true
	}
	existing_token, typecast_ok := session.Values["authenticity_token"].(string)
	if typecast_ok && len(existing_token) > 0 {
		has_token = true
	}

	if !has_token {
		payload := time.Now().UTC().Unix()
		token := Sha256(strconv.FormatInt(payload, 10))
		authenticity_token, token_err := f_s_encrypt_string(fmt.Sprintf("%d|%v", payload, token), *flag_s_session_authenticity_token_secret)
		if token_err != nil {
			log.Printf("failed to generate an authenticity token due to err %v", token_err)
		}
		session.Values["authenticity_token"] = authenticity_token
	}

}

func r_get_login(c *gin.Context) {

}

type TSSessionAlert struct {
	Kind        string    `json:"kind"`
	Message     string    `json:"message"`
	ExpiresAt   time.Time `json:"expires_at"`
	ClearOnRead bool      `json:"clear_on_read"`
}

func NewSessionAlert(kind string, message string, expires_at time.Time, clear_on_read bool) *TSSessionAlert {
	return &TSSessionAlert{
		Kind:        kind,
		Message:     message,
		ExpiresAt:   expires_at,
		ClearOnRead: clear_on_read,
	}
}

type TSFlashMessages map[time.Time]*TSSessionAlert

func f_session_flash(session *sessions.Session, alert *TSSessionAlert) {
	session.Values["flash"] = TSFlashMessages{time.Now().UTC(): alert}
}

func f_enforce_authenticity_token(c *gin.Context) {
	session := g_get_session(c)
	challenge_authenticity_token := c.PostForm("authenticity_token")
	b_permit_access := true
	s_error_message := ""
	if len(challenge_authenticity_token) != 64 {
		b_permit_access = false
		s_error_message = "oh you think you're clever don't you?"
	}
	authenticity_token, typecast_err := session.Values["authenticity_token"].(string)
	if !typecast_err || challenge_authenticity_token != authenticity_token {
		b_permit_access = false
		s_error_message = "watch your six"
	}
	if !b_permit_access {
		f_session_flash(session, NewSessionAlert("error", s_error_message, time.Now().Add(36*time.Second), true))
		log.Printf("f_enforce_authenticity_token error %v", s_error_message)
		f_add_ip_to_watch_list(net.ParseIP(f_s_filtered_ip(c)))
	}

	c.Next()
}

func f_username_to_identifier(username string) ([64]byte, error) {
	identifier := Sha256(username)
	if len(identifier) != 64 {
		return [64]byte{}, fmt.Errorf("identifier must be exactly 64 characters long")
	}

	var arr [64]byte
	copy(arr[:], username)
	return arr, nil
}

func middleware_ensure_authenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := f_gin_check_authenticated_in_session(c)
		if err != nil {
			session := g_get_session(c)
			var last_failed_attempt time.Time
			_, last_failed_attempts_exist := session.Values["last_failed_login_attempt"]
			if !last_failed_attempts_exist {
				f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
				referer := c.GetHeader("Referer")
				if len(referer) > 0 {
					c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
					return
				}
				c.Redirect(http.StatusOK, "/login")
				return
			}

			var valid_format bool
			last_failed_attempt, valid_format = session.Values["last_failed_login_attempt"].(time.Time)
			if !valid_format {
				session.Values["last_failed_login_attempt"] = time.Now().UTC()
			}

			var failed_attempts int
			var next_failed_attempts int
			_, failed_attempts_exists := session.Values["failed_attempts"]
			if failed_attempts_exists {
				last_value, valid_value := session.Values["failed_attempts"].(int)
				if !valid_value {
					session.Values["failed_attempts"] = 1
				} else {
					failed_attempts = last_value
					next_failed_attempts = last_value + 1
					session.Values["failed_attempts"] = next_failed_attempts
				}
			}

			maybe_username, username_exists := session.Values["username"]
			if !username_exists {
				f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
				referer := c.GetHeader("Referer")
				if len(referer) > 0 {
					c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
					return
				}
				c.Redirect(http.StatusOK, "/login")
				return
			}

			username, is_username := maybe_username.(string)
			if !is_username {
				f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
				referer := c.GetHeader("Referer")
				if len(referer) > 0 {
					c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
					return
				}
				c.Redirect(http.StatusOK, "/login")
				return
			}

			account, load_err := load_username_account(username)
			if load_err != nil {
				f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
				referer := c.GetHeader("Referer")
				if len(referer) > 0 {
					c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
					return
				}
				c.Redirect(http.StatusOK, "/login")
				return
			}

			account.LastFailedLogin = time.Now().UTC()
			account.LastFailedLoginIP = net.IP(f_s_client_ip(c.Request))
			account.FailedLoginAttempts = failed_attempts
			err := store_username_account(account)
			if err != nil {
				log.Printf("failed to store_username_account for account %v", account)
			}

			if time.Since(last_failed_attempt) < time.Duration(fibonacci(failed_attempts))*time.Second*3 {
				if failed_attempts >= *flag_i_auth_max_failed_logins {
					account.Locked = true
					account.LockedAt = time.Now().UTC()
					account.LockedByIP = net.IP(f_s_client_ip(c.Request))
					err := store_username_account(account)
					if err != nil {
						log.Printf("failed to store_username_account for account %v", account)
					}
				}
				f_session_flash(session, NewSessionAlert("error", fmt.Sprintf("You must wait %.0f seconds before trying to sign in again.", time.Since(last_failed_attempt).Seconds()), time.Now().Add(45*time.Second), true))
				c.Redirect(http.StatusOK, "/login")
				return
			}

			f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
			referer := c.GetHeader("Referer")
			if len(referer) > 0 {
				c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
				return
			}
			c.Redirect(http.StatusOK, "/login")
			return
		}
		c.Next()
	}
}

func f_gin_check_authenticated_in_session(c *gin.Context) error {
	ip := f_s_client_ip(c.Request)
	session := g_get_session(c)
	authenticated, authenticated_value_set := session.Values["authenticated"].(bool)
	last_sign_in_ip, last_sign_in_ip_value_set := session.Values["last_sign_in_ip"].(string)
	_, last_sign_in_at_value_set := session.Values["last_sign_in_at"].(time.Time)
	_, username_value_set := session.Values["username"].(string)

	if authenticated_value_set && last_sign_in_ip_value_set && last_sign_in_at_value_set && username_value_set {
		if authenticated && last_sign_in_ip == ip {
			return nil
		}
	}
	return fmt.Errorf("session not authenticated")
}

func r_post_login(c *gin.Context) {
	session := g_get_session(c)
	username := c.PostForm("username")
	identifier, identifier_err := f_username_to_identifier(username)
	if identifier_err != nil {
		f_session_flash(session, NewSessionAlert("error", "Cannot validate username.", time.Now().Add(36*time.Second), true))
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	account, user_load_err := load_username_account(username)
	if user_load_err != nil {
		f_session_flash(session, NewSessionAlert("error", "No such username.", time.Now().Add(36*time.Second), true))
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	raw_password := c.PostForm("password")
	err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(raw_password))
	if err != nil {
		f_session_flash(session, NewSessionAlert("error", "Invalid password.", time.Now().Add(36*time.Second), true))
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}
	session.Values["authenticated"] = true
	session.Values["last_sign_in_ip"] = f_s_client_ip(c.Request)
	session.Values["last_sign_in_at"] = time.Now().UTC()
	session.Values["username"] = account.Username
	checksum_path, path_err := checksum_to_path(string(identifier[:]))
	if path_err != nil {
		f_session_flash(session, NewSessionAlert("error", "Encountered a problem.", time.Now().Add(45*time.Second), true))
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	}
	session.Values["account_db_path"] = filepath.Join(*flag_s_users_database_directory, checksum_path, account_database_filename)
	f_session_flash(session, NewSessionAlert("success", fmt.Sprintf("Welcome back %v", account.Username), time.Now().Add(63*time.Second), true))
	referer := c.GetHeader("Referer")
	if len(referer) > 0 {
		c.Redirect(http.StatusOK, referer)
	}
	c.Redirect(http.StatusOK, "/")
	return
}

func r_get_logout(c *gin.Context) {

}

func r_get_register(c *gin.Context) {

}

func r_post_register(c *gin.Context) {

}

func r_get_challenge_password(c *gin.Context) {

}

func r_post_challenge_password(c *gin.Context) {

}

func r_get_forgot_password(c *gin.Context) {

}

func r_post_forgot_password(c *gin.Context) {

}

func r_get_change_email(c *gin.Context) {

}

func r_post_change_email(c *gin.Context) {

}

func r_get_manage_profile(c *gin.Context) {

}

func r_get_public_profile(c *gin.Context) {

}

func r_get_account_locked(c *gin.Context) {

}

func r_get_account_banned(c *gin.Context) {

}

func r_get_download_account_data(c *gin.Context) {

}

func r_post_download_account_data(c *gin.Context) {

}

func r_get_request_account_deletion(c *gin.Context) {

}

func r_post_request_account_deletion(c *gin.Context) {

}

func worker_delete_requested_accounts() {

}

// worker_audit_user_security is a scheduled job that runs at an interval that analyzes the users data directory
// and finds anomalies with the data and then enqueues messages that the user will need to respond to.
func worker_audit_user_security() {

}

var (
	sem_users_database_open_files = go_sema.New(1776) // permit 1776 to open files
)

func (tsudar *ts_user_database_account_record) Save() error {
	return store_username_account(tsudar)
}

func store_username_messages(username string, messages *ts_user_database_messages_record) error {
	checksum := Sha256(username)
	var identifier [64]byte
	copy(identifier[:], checksum)
	return store_account_messages(identifier, messages)
}

func store_account_messages(identifier [64]byte, messages *ts_user_database_messages_record) error {
	checksum := string(identifier[:])
	checksum_path, path_err := checksum_to_path(checksum)
	if path_err != nil {
		return path_err
	}
	path := filepath.Join(*flag_s_users_database_directory, checksum_path)
	info, info_err := os.Stat(path)
	if errors.Is(os.ErrNotExist, info_err) {
		err := os.MkdirAll(path, 0700)
		if err != nil {
			return err
		}
	}
	if info.IsDir() && len(info.Name()) != 19 {
		// last directory in the path
		return fmt.Errorf("improperly formatted directory path before storing account.json")
	}

	err := write_any_to_file(path, messages_database_filename, messages)
	if err != nil {
		return err
	}
	return nil
}

func load_account_messages(identifier [64]byte) (*ts_user_database_messages_record, error) {
	checksum := Sha256(string(identifier[:]))
	checksum_path, path_err := checksum_to_path(checksum)
	if path_err != nil {
		return nil, path_err
	}
	path := filepath.Join(*flag_s_users_database_directory, checksum_path)
	info, info_err := os.Stat(path)
	if info_err != nil {
		return nil, info_err
	}
	if info.IsDir() && len(info.Name()) == 19 {
		// end of the path
		ok, err := check_path_checksum(path, checksum)
		if !ok || err != nil {
			return nil, err
		}
	}
	messages_path := filepath.Join(path, messages_database_filename)
	sem_users_database_open_files.Acquire()
	messages_bytes, bytes_err := os.ReadFile(messages_path)
	sem_users_database_open_files.Release()
	if bytes_err != nil {
		return nil, bytes_err
	}
	var messages ts_user_database_messages_record
	unmarshal_err := json.Unmarshal(messages_bytes, &messages)
	if unmarshal_err != nil {
		return nil, unmarshal_err
	}
	// memory cleanup
	checksum = ""
	checksum_path = ""
	path = ""
	info = nil
	messages_path = ""
	messages_bytes = nil
	unmarshal_err = nil
	// return pointer to the account data
	return &messages, nil
}

func store_username_account(account *ts_user_database_account_record) error {
	checksum := string(account.Identifier[:])
	checksum_path, path_err := checksum_to_path(checksum)
	if path_err != nil {
		return path_err
	}
	path := filepath.Join(*flag_s_users_database_directory, checksum_path)
	info, info_err := os.Stat(path)
	if errors.Is(os.ErrNotExist, info_err) {
		err := os.MkdirAll(path, 0700)
		if err != nil {
			return err
		}
	}
	if info.IsDir() && len(info.Name()) != 19 {
		// last directory in the path
		return fmt.Errorf("improperly formatted directory path before storing account.json")
	}

	err := write_any_to_file(path, account_database_filename, account)
	if err != nil {
		return err
	}
	return nil
}

func load_username_account(username string) (*ts_user_database_account_record, error) {
	checksum := Sha256(username)
	checksum_path, path_err := checksum_to_path(checksum)
	if path_err != nil {
		return nil, path_err
	}
	path := filepath.Join(*flag_s_users_database_directory, checksum_path)
	info, info_err := os.Stat(path)
	if info_err != nil {
		return nil, info_err
	}
	if info.IsDir() && len(info.Name()) == 19 {
		// end of the path
		ok, err := check_path_checksum(path, checksum)
		if !ok || err != nil {
			return nil, err
		}
	}
	account_path := filepath.Join(path, account_database_filename)
	sem_users_database_open_files.Acquire()
	account_bytes, bytes_err := os.ReadFile(account_path)
	sem_users_database_open_files.Release()
	if bytes_err != nil {
		return nil, bytes_err
	}
	var account ts_user_database_account_record
	unmarshal_err := json.Unmarshal(account_bytes, &account)
	if unmarshal_err != nil {
		return nil, unmarshal_err
	}
	checksum_bytes := [64]byte{}
	bytes_copied := copy(checksum_bytes[:], checksum)
	if bytes_copied != 64 {
		return nil, fmt.Errorf("copy(checksum_bytes[:], checksum) only copied %d bytes. needed 64 bytes", bytes_copied)
	}
	if account.Identifier != checksum_bytes {
		return nil, fmt.Errorf("failed to verify integrity of username %v database path %v", username, path)
	}
	// memory cleanup
	checksum = ""
	checksum_path = ""
	path = ""
	info = nil
	account_path = ""
	account_bytes = nil
	unmarshal_err = nil
	checksum_bytes = [64]byte{}
	// return pointer to the account data
	return &account, nil
}

const usernames_database_filename = "usernames.json"
const account_database_filename = "account.json"
const messages_database_filename = "messages.json"

func check_username(username string) (bool, error) {
	var synced = atomic.Bool{}
	m_usernames.mu.RLock()
	if time.Since(m_usernames.LastFilesystemSync).Seconds() > 30 {
		usernames_bytes, bytes_err := json.Marshal(m_usernames)
		if bytes_err != nil {
			m_usernames.mu.RUnlock()
			return false, bytes_err
		}
		err := write_any_to_file(*flag_s_users_database_directory, usernames_database_filename, usernames_bytes)
		if err != nil {
			m_usernames.mu.RUnlock()
			return false, err
		}
		synced.Store(true)
	}
	_, exists := m_usernames.Usernames[username]
	m_usernames.mu.RUnlock()
	if synced.Load() {
		m_usernames.LastFilesystemSync = time.Now()
	}
	return exists, nil
}

// ts_usernames stored as ./users.db/usernames.json
type ts_usernames struct {
	Usernames          map[string]time.Time // map[Username]RegisteredAt
	LastFilesystemSync time.Time
	mu                 *sync.RWMutex
}

var m_usernames = &ts_usernames{
	mu:        &sync.RWMutex{},
	Usernames: make(map[string]time.Time),
}

// ts_user_database_account_history stored as ./users.db/<identifier>/history.json
type ts_user_database_account_history struct {
	IPs    map[time.Time]net.IP
	Emails map[time.Time]string
}

// ts_user_database_account_record stored as ./users.db/<identifier>/account.json
type ts_user_database_account_record struct {
	Identifier          [64]byte
	Email               string
	Username            string
	Password            string
	RegistrationIP      net.IP
	RegisteredAt        time.Time
	LastLogin           time.Time
	LastFailedLogin     time.Time
	LastFailedLoginIP   net.IP
	Locked              bool
	LockedAt            time.Time
	LockedByIP          net.IP
	FailedLoginAttempts int
}

// ts_user_database_messages_message is stored inside ts_user_database_messages_record called Messages which is a slice
type ts_user_database_messages_message struct {
	SentAt       time.Time `json:"sent_at"`
	ReadAt       time.Time `json:"read_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	DeleteOnRead bool      `json:"delete_on_read"`
	Subject      string    `json:"subject"`
	Kind         string    `json:"kind"` // valid options are { success }, { info }, { warning }, { error }, and { fatal }
	Body         string    `json:"body"`
}

// ts_user_database_messages_record stored as ./users.db/<identifier>/messages.json
type ts_user_database_messages_record struct {
	Messages []ts_user_database_messages_message `json:"messages"`
}

func new_user_database(email string) {

}
