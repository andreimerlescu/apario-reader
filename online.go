package main

import (
	`context`
	`log`
	`time`
)

func clean_online_counter_scheduler(ctx context.Context) {
	duration := time.Duration(*flag_i_online_refresh_delay_minutes) * time.Second * 9 // 9x scalar required (17 minutes -> seconds * 9 = 2m33s to flush offline users)
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Duration(*flag_i_online_refresh_delay_minutes) * time.Minute // match configuration language
			mu_online_list.RLock()
			entries := m_online_list
			mu_online_list.RUnlock()
			for _, entry := range entries {
				now := time.Now().UTC()
				if now.After(entry.DeleteOn) {
					mu_online_list.Lock()
					delete(m_online_list, entry.IP.String())
					mu_online_list.Unlock()
				} else {
					log.Printf("%s %s %s %v", "entry.LastAction", entry.LastAction, " is less than cutoff ", cutoff)
				}
			}
		}
	}
}

func load_online_counter_cache_scheduler(ctx context.Context) {
	duration := time.Duration(*flag_i_online_refresh_delay_minutes) * time.Second * 3 // triple scalar required (17 minutes of online activity = 51s refresh time of online list)
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mu_online_list.RLock()
			a_i_cached_online_counter.Store(int64(len(m_online_list)))
			mu_online_list.RUnlock()
		}
	}
}
