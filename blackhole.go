package main

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func TheBlackHole() *BlackHole {
	if a_blackhole == nil {
		a_blackhole = &BlackHole{
			Entropy: make(map[string]DoomedStar),
		}
	}
	return a_blackhole
}

type BlackHole struct {
	Entropy map[string]DoomedStar
	Locker  sync.RWMutex
}

// Unique removes duplicates from a slice of comparable type T.
// It returns a new slice containing unique elements in the original order.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(slice))

	for _, item := range slice {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func (bh *BlackHole) Observe(ctx context.Context) {
	persister := time.NewTicker(15 * time.Minute)
	reloader := time.NewTicker(5 * time.Minute)
	defer persister.Stop()
	defer reloader.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-persister.C:
			err := bh.Persist()
			if err != nil {
				log.Println("error persisting blackhole:", err)
			}
		case <-reloader.C:
			err := bh.Reload()
			if err != nil {
				log.Println("error reloading blackhole:", err)
			}
		}
	}
}

func (bh *BlackHole) Reload() error {
	jsonBytes, err := os.ReadFile(*flag_s_blackhole_file_path)
	if err != nil {
		return err
	}
	var target []DoomedStar
	err = json.Unmarshal(jsonBytes, &target)
	if err != nil {
		return err
	}
	bh.Locker.Lock()
	defer bh.Locker.Unlock()
	if len(bh.Entropy) != len(target) {
		for _, star := range target {
			if doomedStar, exists := bh.Entropy[star.IP]; exists {
				star.FailedPaths = append(star.FailedPaths, doomedStar.FailedPaths...)
				star.FailedPaths = Unique(star.FailedPaths)
				bh.Entropy[star.IP] = star
			} else {
				bh.Entropy[star.IP] = star
			}
		}
	}
	return nil
}

func (bh *BlackHole) Persist() error {
	bh.Locker.Lock()
	defer bh.Locker.Unlock()
	jsonBytes, jsonErr := json.MarshalIndent(bh.Entropy, "", "  ")
	if jsonErr != nil {
		return jsonErr
	}
	return os.WriteFile(*flag_s_blackhole_file_path, jsonBytes, 0644)
}

func (bh *BlackHole) ReceiveGinContext(c *gin.Context) {
	ipStr := f_s_filtered_ip(c)
	ip, _ := net.ResolveIPAddr("ip", ipStr)
	path := c.Request.URL.String()
	now := time.Now().UTC()
	bh.Locker.Lock()
	defer bh.Locker.Unlock()
	var skipped []DoomedStar
	doomedStar := new(DoomedStar)
	for _, star := range bh.Entropy {
		if strings.EqualFold(star.IP, ip.String()) {
			*doomedStar = star
		} else {
			skipped = append(skipped, star)
		}
	}
	if doomedStar == nil {
		doomedStar = &DoomedStar{
			IP:          ip.String(),
			FailedPaths: []string{path},
			RequestPath: path,
			DoomedOn:    time.Now().UTC(),
		}
	} else {
		doomedStar.FailedPaths = append(doomedStar.FailedPaths, path)
		doomedStar.RequestPath = path
		doomedStar.UpdatedOn = now
	}
	bh.Entropy[doomedStar.IP] = *doomedStar
}

type DoomedStar struct {
	IP          string
	FailedPaths []string
	RequestPath string
	DoomedOn    time.Time
	UpdatedOn   time.Time
}

var a_blackhole *BlackHole
