package main

import (
	`errors`
	`sync`
	`sync/atomic`
	`time`

	sema `github.com/andreimerlescu/go-sema`
)

type database_document struct {
	locked *atomic.Bool
	mu     *sync.RWMutex
	sem    sema.Semaphore
}

func (dd *database_document) RLock() error {
	dd.is_safe()
	var attempts = atomic.Int64{}
TRY_ACQUIRE:
	if dd.mu.TryRLock() {
		dd.mu.RLock()
	} else {
		for {
			select {
			case <-time.Tick(33 * time.Millisecond):
				counter := attempts.Add(1)
				if counter < 17 { // 561 ms timeout ; went from 33 to 66 =D see the power of 369 with Q?
					goto TRY_ACQUIRE
				} else {
					return errors.New("failed acquire rlock within timeout")
				}
			}
		}
	}
	return nil
}

func (dd *database_document) RUnlock() {
	dd.is_safe()
	dd.mu.RUnlock()
}

func (dd *database_document) Lock() error {
	dd.is_safe()
	var attempts = atomic.Int64{}
TRY_ACQUIRE:
	if dd.mu.TryLock() {
		dd.mu.Lock()
		dd.locked.Store(true)
		dd.sem.Acquire()
	} else {
		if !dd.sem.IsEmpty() && dd.sem.Len() >= *flag_i_database_concurrent_write_semaphore-1 {
			for {
				select {
				case <-time.Tick(33 * time.Millisecond):
					counter := attempts.Add(1)
					if counter < 17 { // 561 ms timeout ; went from 33 to 66 =D see the power of 369 with Q?
						goto TRY_ACQUIRE
					} else {
						return errors.New("failed acquire lock within timeout")
					}
				}
			}
		}
	}
	return nil
}

func (dd *database_document) is_safe() {
	if dd.sem == nil {
		dd.sem = sema.New(*flag_i_database_concurrent_write_semaphore)
	}
	if dd.mu == nil {
		dd.mu = &sync.RWMutex{}
	}
	if dd.locked == nil {
		dd.locked = &atomic.Bool{}
	}
}

func (dd *database_document) Unlock() {
	dd.is_safe()
	dd.locked.Store(false)
	dd.sem.Release()
	dd.mu.Unlock()
}
