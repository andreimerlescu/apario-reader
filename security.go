package main

import (
	`context`
	`crypto/sha256`
	`encoding/hex`
	`log`
	`net`
	`strings`
	`sync`
	`sync/atomic`
	`time`

	`github.com/gin-gonic/gin`
)

func f_patch_server_with_banned_ip(ip net.IP) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
	}()
	sem_banned_ip_patch.Acquire()
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

func f_add_ip_to_watch_list(ip net.IP) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
	}()
	mu_ip_watch_list.RLock()
	counter, found := m_ip_watch_list[ip.String()]
	mu_ip_watch_list.RUnlock()
	if !found {
		mu_ip_watch_list.Lock()
		m_ip_watch_list[ip.String()] = &atomic.Int64{}
		counter = m_ip_watch_list[ip.String()]
		mu_ip_watch_list.Unlock()
	}
	new_count := counter.Add(1)

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
	}
	return ip
}

func f_schedule_ip_ban_list_cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(*flag_i_ip_ban_list_synchronization) * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f_perform_ip_ban_list_cleanup(ctx)

		}
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
		mu_ip_watch_list.RLock()
		counter, found := m_ip_watch_list[ip.String()]
		mu_ip_watch_list.RUnlock()
		if !found {
			mu_ip_watch_list.Lock()
			m_ip_watch_list[ip.String()] = &atomic.Int64{}
			counter = m_ip_watch_list[ip.String()]
			mu_ip_watch_list.Unlock()
		}
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
