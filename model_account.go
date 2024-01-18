package main

import (
	`context`
	`encoding/json`
	`errors`
	`fmt`
	`io/fs`
	`log`
	`net`
	`os`
	`path/filepath`
	`strings`
	`time`
)

// TSAccount is saved at <users.db>/<identifier-path>/account.json
type TSAccount struct {
	database_document
	Identifier          string    `json:"identifier"`
	Version             DDVersion `json:"version"`
	Username            string    `json:"username"`
	Password            string    `json:"password"`
	Email               string    `json:"email"`
	RegistrationIP      net.IP    `json:"registration_ip"`
	RegisteredAt        time.Time `json:"registered_at"`
	LastLogin           time.Time `json:"last_login"`
	LastFailedLogin     time.Time `json:"last_failed_login"`
	LastFailedLoginIP   net.IP    `json:"last_failed_login_ip"`
	Locked              bool      `json:"locked"`
	LockedAt            time.Time `json:"locked_at"`
	LockedByIP          net.IP    `json:"locked_by_ip"`
	FailedLoginAttempts int       `json:"failed_login_attempts"`
}

func perform_username_sync() error {
	return filepath.Walk(*flag_s_users_database_directory, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil // skip over files
		}

		cleaned_path := strings.ReplaceAll(path, *flag_s_users_database_directory, ``)
		maybe_identifier := strings.ReplaceAll(cleaned_path, string(os.PathSeparator), ``)
		identifier, maybe_err := ParseIdentifier(maybe_identifier)
		if maybe_err != nil {
			return nil
		}
		// valid identifier provided
		account_json_path := filepath.Join(path, "account.json")
		json_info, info_err := os.Stat(account_json_path)
		if info_err != nil {
			return nil
		}

		if json_info.Size() < 3 {
			return nil
		}

		a := &TSAccount{Identifier: identifier.String()}
		err = a.Load()
		if err != nil {
			return nil
		}

		if a.Identifier == identifier.String() {
			a.SyncUsername()
		}

		return nil
	})
}

func sync_user_directory(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 3)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := perform_username_sync()
			if err != nil {
				log.Printf("sync_user_directory timer triggered perform_username_sync err %v", err)
			}
		}
	}
}

func GetAccount(username string) (*TSAccount, error) {
	// is it in m_usernames
	m_usernames.mu.RLock()
	identifier, exists := m_usernames.UsernameIdentifiers[username]
	m_usernames.mu.RUnlock()
	if exists {
		a := &TSAccount{Identifier: identifier, Username: username}
		err := a.Load()
		if err != nil {
			return nil, err
		}
		return a, nil
	}

	return nil, errors.New("no such account")
}

func (a *TSAccount) Load() error {
	err := a.database_document.Lock()
	if err != nil {
		return err
	}
	defer a.database_document.Unlock()

	if len(a.Identifier) == 0 {
		return errors.New("identifier missing")
	}

	db_path := filepath.Join(*flag_s_users_database_directory, identifier_to_path(a.Identifier))
	payload, payload_err := read_payload_from_database(filepath.Join(db_path, "account.json"))
	if payload_err != nil {
		return payload_err
	}
	maybe_account, is_account := payload.(TSAccount)
	if !is_account {
		return errors.New("payload on disk for account.json not of type TSAccount")
	}
	if maybe_account.Identifier != a.Identifier {
		return errors.New("invalid identifier found, cannot load from disk")
	}
	a.Version = maybe_account.Version
	a.Username = maybe_account.Username
	a.SyncUsername()
	a.Password = maybe_account.Password
	a.Email = maybe_account.Email
	a.RegisteredAt = maybe_account.RegisteredAt
	a.RegistrationIP = maybe_account.RegistrationIP
	a.LastLogin = maybe_account.LastLogin
	a.LastFailedLogin = maybe_account.LastFailedLogin
	a.LastFailedLoginIP = maybe_account.LastFailedLoginIP
	a.Locked = maybe_account.Locked
	a.LockedAt = maybe_account.LockedAt
	a.LockedByIP = maybe_account.LockedByIP
	a.FailedLoginAttempts = maybe_account.FailedLoginAttempts
	return nil
}

func (a *TSAccount) SyncUsername() {
	m_usernames.mu.Lock()
	m_usernames.IdentifierVersions[a.Identifier] = a.Version.String()
	m_usernames.UsernameJoins[a.Identifier] = a.RegisteredAt
	m_usernames.IdentifierUsernames[a.Identifier] = a.Username
	m_usernames.UsernameIdentifiers[a.Username] = a.Identifier
	m_usernames.mu.Unlock()
	_, err := check_username(a.Username)
	if err != nil {
		log.Printf("TSAccount SyncUsername() failed to check_username(%v) due to err %v", a.Username, err)
	}
}

func (a *TSAccount) Save() error {
	err := a.database_document.Lock()
	if err != nil {
		return err
	}
	defer a.database_document.Unlock()

	if len(a.Identifier) == 0 {
		identifier, identifier_err := NewIdentifier(*flag_s_users_database_directory, *flag_i_user_identifier_length, 0, 30)
		if identifier_err != nil {
			return identifier_err
		}
		a.Identifier = identifier.String()
	}

	a.SyncUsername()

	// account version control
	// for document_version file will be = arg2=[<database>/<identifier-path>]/versions/arg1=[<version>].json
	db_path := filepath.Join(*flag_s_users_database_directory, identifier_to_path(a.Identifier))
	version, version_err := version_exists_in_database_path(a.Version.String(), db_path)
	if version_err != nil {
		log.Printf("%v", version_err)
	}

	// no version exists on disk, lets save this document to the disk
	if version == nil {
		// no version exists on disk
		// perform a version bump and backup
		dd := database_document{}
		dd.is_safe() // ensure that .Save() can run
		bytes, bytes_err := json.Marshal(a)
		if bytes_err != nil {
			return bytes_err
		}
		checksum := Sha256(string(bytes))
		av := &AccountVersion{
			database_document: dd,
			Identifier:        a.Identifier,
			DateCreated:       time.Now().UTC(),
			Checksum:          checksum,
			Version:           a.Version,
			Account:           *a,
		}
		av_err := av.Save() // persist struct as json to disk
		if av_err != nil {
			log.Printf("failed to save the AccountVersion due to err %v", av_err)
			return av_err
		}
	}
	return write_to_file(filepath.Join(*flag_s_users_database_directory, identifier_to_path(a.Identifier), "account.json"), a)
}

// AccountVersion is stored at <users.db>/<identifier-to-path>/versions/<version>.json
type AccountVersion struct {
	database_document
	Identifier  string    `json:"identifier"`
	DateCreated time.Time `json:"date_created"`
	Checksum    string    `json:"checksum"`
	Version     DDVersion `json:"version"`
	Account     TSAccount `json:"account"`
}

func (av *AccountVersion) Save() error {
	err := av.database_document.Lock()
	if err != nil {
		return err
	}
	defer av.database_document.Unlock()
	return write_to_file(filepath.Join(*flag_s_database, identifier_to_path(av.Identifier), "versions", fmt.Sprintf("%s.json", av.Version.String())), av)
}
