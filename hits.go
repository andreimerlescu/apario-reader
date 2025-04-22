package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Hit struct {
	Total int64
}

func (h *Hit) Load() int64 {
	return h.Total
}

func (h *Hit) Add(i int64) int64 {
	return atomic.AddInt64(&h.Total, i)
}

func (h *Hit) Sub(i int64) int64 {
	return atomic.AddInt64(&h.Total, -i)
}

func (h *Hit) Store(i int64) int64 {
	atomic.StoreInt64(&h.Total, i)
	return h.Total
}

var hit_jar *Hit

func Hits() *Hit {
	if hit_jar == nil {
		hit_jar = &Hit{
			Total: 0,
		}
		_ = hit_jar.Reload()
	}
	return hit_jar
}

func middleware_count_hits() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_count_hits(c)
		c.Next()
	}
}

func f_i_hits() int64 {
	return Hits().Load()
}

func (h *Hit) Persist() error {
	hitFile, openErr := os.OpenFile(*flag_s_hits_file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if openErr != nil {
		return fmt.Errorf("error opening IP ban file for writing: %v", openErr)
	}
	defer hitFile.Close()

	encoder := json.NewEncoder(hitFile)
	if err := encoder.Encode(Hits()); err != nil {
		return fmt.Errorf("error encoding Hits: %v", err)
	}
	return nil
}

func (h *Hit) Reload() error {
	hitFile, openErr := os.OpenFile(*flag_s_hits_file, os.O_RDONLY, 0600)
	if openErr != nil {
		return fmt.Errorf("error opening Hits: %v", openErr)
	}
	defer hitFile.Close()

	var results Hit
	decoder := json.NewDecoder(hitFile)
	if err := decoder.Decode(&results); err != nil {
		return fmt.Errorf("error decoding Hits: %v", err)
	}
	Hits().Store(results.Total)
	return nil
}

func persist_hits_offline(ctx context.Context) {
	tickerPersist := time.NewTicker(369 * time.Second)
	tickerReload := time.NewTicker(434 * time.Second)
	defer tickerPersist.Stop()
	defer tickerReload.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerPersist.C: // Save the hits to disk every 369 seconds
			err := Hits().Persist()
			if err != nil {
				log.Printf("Error persisting hits: %v", err)
			}
		case <-tickerReload.C: // reload hits from disk every 3 minutes
			err := Hits().Reload()
			if err != nil {
				log.Printf("Error reloading hits: %v", err)
			}
		}
	}
}

func handler_count_hits(c *gin.Context) {
	Hits().Add(1)
}
