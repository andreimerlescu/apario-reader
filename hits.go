package main

import (
	`context`
	`encoding/json`
	`log`
	`os`
	`sync/atomic`
	`time`

	`github.com/gin-gonic/gin`
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

func persist_hits_offline(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 30): // Save the hits to disk every 30 seconds
			hitFile, openErr := os.OpenFile(*flag_s_hits_file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			if openErr != nil {
				log.Printf("Error opening IP ban file for writing: %v", openErr)
				return
			}
			defer hitFile.Close()

			encoder := json.NewEncoder(hitFile)
			if err := encoder.Encode(Hits()); err != nil {
				log.Printf("Error encoding Hits: %v", err)
			}

		case <-time.After(time.Minute * 3): // reload hits from disk every 3 minutes
			hitFile, openErr := os.OpenFile(*flag_s_hits_file, os.O_RDONLY, 0600)
			if openErr != nil {
				log.Printf("Error opening Hits: %v", openErr)
				return
			}
			defer hitFile.Close()

			var results Hit
			decoder := json.NewDecoder(hitFile)
			if err := decoder.Decode(&results); err != nil {
				log.Printf("Error decoding Hits: %v", err)
				return
			}
			Hits().Store(results.Total)
		}
	}
}

func handler_count_hits(c *gin.Context) {
	Hits().Add(1)
}
