package main

import (
	`encoding/json`
	`sync`
	`sync/atomic`
	`time`
)

const usernames_database_filename = "usernames.json"
const account_database_filename = "account.json"

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
	_, exists := m_usernames.UsernameJoins[username]
	m_usernames.mu.RUnlock()
	if synced.Load() {
		m_usernames.LastFilesystemSync = time.Now()
	}
	return exists, nil
}

var AllUsernames map[string]string // map[Username]Identifier
var mu_AllUsernames = &sync.RWMutex{}

// ts_usernames stored as ./users.db/usernames.json
type ts_usernames struct {
	UsernameJoins       map[string]time.Time // map[Username]RegisteredAt
	UsernameIdentifiers map[string]string    // map[Username]Identifier
	IdentifierUsernames map[string]string    // map[Identifier]Username
	IdentifierVersions  map[string]string    // map[Identifier]Version
	LastFilesystemSync  time.Time
	mu                  *sync.RWMutex
}

var m_usernames = &ts_usernames{
	mu:                  &sync.RWMutex{},
	UsernameJoins:       make(map[string]time.Time),
	UsernameIdentifiers: make(map[string]string),
	IdentifierUsernames: make(map[string]string),
	IdentifierVersions:  make(map[string]string),
}
