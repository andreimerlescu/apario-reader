package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

func f_patch_server_with_banned_ip(ip net.IP) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
	}()
	requestedAt := time.Now().UTC()
	sem_banned_ip_patch.Acquire()
	if since := time.Since(requestedAt).Seconds(); since > 1.7 {
		log.Printf("took %.0f seconds to acquire sem_banned_ip_patch queue position", since)
	}
	defer sem_banned_ip_patch.Release()
	// TODO: add the option to use firewall-cmd, ufw or iptables to block the IP address from the server with a comment
	log.Printf("need to patch the server with banning the ip %v", ip)
}

func f_add_ip_to_ban_list(ip net.IP) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
	}()
	mu_ip_ban_list.Lock()
	defer mu_ip_ban_list.Unlock()

	m_ip_ban_list = append(m_ip_ban_list, ip)
	go f_patch_server_with_banned_ip(ip)
}

// f_add_ip_to_watch_list adds ip to m_ip_watch_list with mu_ip_watch_list
func f_add_ip_to_watch_list(ip net.IP) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
	}()

	mu_ip_watch_list.Lock()
	counter, found := m_ip_watch_list[ip.String()]
	if !found {
		m_ip_watch_list[ip.String()] = &atomic.Int64{}
		counter = m_ip_watch_list[ip.String()]
	}
	new_count := counter.Add(1)
	mu_ip_watch_list.Unlock()

	if new_count >= 6 {
		f_add_ip_to_ban_list(ip)
	}
}

func f_ip_in_ban_list(ip net.IP) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
	}()
	mu_ip_ban_list.RLock()
	defer mu_ip_ban_list.RUnlock()

	for _, banned_ip := range m_ip_ban_list {
		if ip.Equal(banned_ip) {
			return true
		}
	}
	return false
}

func f_s_filtered_ip(c *gin.Context) string {
	var ip string
	clientIP := c.ClientIP()
	forwardedIP := f_s_client_ip(c.Request)
	if len(clientIP) != 0 && len(forwardedIP) != 0 && strings.Contains(forwardedIP, ":") && !strings.Contains(clientIP, ":") {
		ip = c.ClientIP()
	} else if len(clientIP) != 0 && len(forwardedIP) != 0 && !strings.Contains(forwardedIP, ":") && strings.Contains(clientIP, ":") {
		ip = forwardedIP
	} else if len(clientIP) == 0 && len(forwardedIP) != 0 {
		ip = forwardedIP
	} else if len(forwardedIP) == 0 && len(c.ClientIP()) != 0 {
		ip = c.ClientIP()
	} else if len(forwardedIP) == 0 && len(c.ClientIP()) == 0 {
		ip = ""
	} else {
		if strings.EqualFold(forwardedIP, clientIP) {
			ip = strings.Clone(clientIP)
		} else {
			ip = strings.Clone(forwardedIP)
		}
	}
	return ip
}

func f_schedule_ip_ban_list_cleanup(ctx context.Context) {
	ticker1 := time.NewTicker(time.Duration(*flag_i_ip_ban_list_synchronization) * time.Second)
	ticker2 := time.NewTicker(3 * time.Minute) // every 3 minutes save to disk
	ticker3 := time.NewTicker(6 * time.Minute) // every 6 minutes restore from disk
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker1.C:
			f_perform_ip_ban_list_cleanup(ctx)
		case <-ticker2.C:
			f_perform_ip_ban_fs_sync(ctx)
		case <-ticker3.C:
			f_perform_ip_ban_fs_load(ctx)
		}
	}
}

func f_perform_ip_ban_fs_load(ctx context.Context) {
	ipFile, openErr := os.OpenFile(*flag_s_ip_ban_file, os.O_RDONLY, 0600)
	if openErr != nil {
		log.Printf("Error opening IP ban file: %v", openErr)
		return
	}
	defer ipFile.Close()

	if ctx.Err() != nil {
		log.Println("Operation cancelled:", ctx.Err())
		return
	}

	var results ts_ip_save
	decoder := json.NewDecoder(ipFile)
	if err := decoder.Decode(&results); err != nil {
		log.Printf("Error decoding IP ban list: %v", err)
		return
	}

	mu_ip_ban_list.Lock()
	defer mu_ip_ban_list.Unlock()
	for key, entry := range results.Entries {
		if ctx.Err() != nil {
			log.Println("Operation cancelled during processing:", ctx.Err())
			return
		}
		ip := net.ParseIP(key)
		if ip != nil {
			m_ip_ban_list = append(m_ip_ban_list, ip)
			updateWatchCounter(ip, entry.Counter)
		}
	}
}

func f_perform_ip_ban_fs_sync(ctx context.Context) {
	mu_ip_ban_list.RLock()
	var results ts_ip_save
	results.Entries = make(map[string]ts_ip_save_entry)
	for _, ip := range m_ip_ban_list {
		if ctx.Err() != nil {
			log.Println("Operation cancelled before file operations:", ctx.Err())
			mu_ip_ban_list.RUnlock()
			return
		}
		ipStr := ip.String()
		results.Entries[ipStr] = ts_ip_save_entry{IP: ip, Counter: getCurrentCounter(ipStr)}
	}
	mu_ip_ban_list.RUnlock()

	ipFile, openErr := os.OpenFile(*flag_s_ip_ban_file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if openErr != nil {
		log.Printf("Error opening IP ban file for writing: %v", openErr)
		return
	}
	defer ipFile.Close()

	encoder := json.NewEncoder(ipFile)
	if err := encoder.Encode(results); err != nil {
		log.Printf("Error encoding IP ban list: %v", err)
	}
}

func f_perform_ip_ban_list_cleanup(ctx context.Context) {
	var results ts_ip_save
	mu_ip_ban_list.RLock()
	ips := m_ip_ban_list
	mu_ip_ban_list.RUnlock()
	duration := time.Duration(*flag_i_ip_ban_list_synchronization) * time.Millisecond
	timeoutCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	breakFor := atomic.Bool{}
	ctxCanceled := atomic.Bool{}
	tryLockCount := atomic.Int64{}

	if len(results.Entries) == 0 || results.Entries == nil {
		results.mu = &sync.RWMutex{}
		breakFor.Store(false)
		for {
			select {
			case <-ctx.Done():
				breakFor.Store(true)
				ctxCanceled.Store(true)
				break
			case <-timeoutCtx.Done():
				breakFor.Store(true)
				break
			case <-time.Tick(10 * time.Millisecond):
				count := tryLockCount.Add(1)
				if count < int64(*flag_i_ip_ban_list_synchronization) && !breakFor.Load() && results.mu.TryLock() { // max 170ms to unlock
					results.mu.Lock()
					results.Entries = make(map[string]ts_ip_save_entry)
					results.mu.Unlock()
					breakFor.Store(true)
					break
				}
			}
			if breakFor.Load() {
				break
			}
		}
		if ctxCanceled.Load() {
			log.Printf("failing because non-timeout ctx was canceled")
			return
		}
	}

	for _, ip := range ips {
		mu_ip_watch_list.Lock()
		counter, found := m_ip_watch_list[ip.String()]
		if !found {
			m_ip_watch_list[ip.String()] = &atomic.Int64{}
			counter = m_ip_watch_list[ip.String()]
		}
		mu_ip_watch_list.Unlock()
		results.mu.Lock()
		results.Entries[ip.String()] = ts_ip_save_entry{
			IP:      ip,
			Counter: counter.Load(),
		}
		results.mu.Unlock()
	}
}

func Sha256(in string) (checksum string) {
	hash := sha256.New()
	hash.Write([]byte(in))
	checksum = hex.EncodeToString(hash.Sum(nil))
	return checksum
}

func updateWatchCounter(ip net.IP, count int64) {
	mu_ip_watch_list.Lock()
	defer mu_ip_watch_list.Unlock()

	ipStr := ip.String()
	counter, found := m_ip_watch_list[ipStr]
	if !found {
		// Initialize the counter if it does not exist
		m_ip_watch_list[ipStr] = new(atomic.Int64)
		counter = m_ip_watch_list[ipStr]
	}
	counter.Store(count) // Set the counter to the specific value
}

func getCurrentCounter(ipStr string) int64 {
	mu_ip_watch_list.RLock()
	defer mu_ip_watch_list.RUnlock()

	if counter, found := m_ip_watch_list[ipStr]; found {
		return counter.Load()
	}
	return 0
}
