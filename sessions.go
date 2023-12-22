package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
	`sync/atomic`
	"time"

	`github.com/gin-contrib/sessions/cookie`
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// RedisStore represents the session store.
type RedisStore struct {
	Codecs  []securecookie.Codec
	Options *sessions.Options // default configuration
	client  *redis.ClusterClient
}

const SESSION_ID string = "PHPSESSID" // intentionally misleading because why not

var (
	x509_cert_pool *x509.CertPool
	store          sessions.Store
	redisAvailable atomic.Bool
	rdb            *redis.ClusterClient
)

func NewRedisStore(client *redis.ClusterClient, keyPairs ...[]byte) *RedisStore {
	return &RedisStore{
		client:  client,
		Codecs:  securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{},
	}
}

func (s *RedisStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(store, name)
}

func (s *RedisStore) New(c *gin.Context, name string) (*sessions.Session, error) {
	session := sessions.NewSession(store, name)
	session.Options = s.Options
	session.IsNew = true

	var err error
	if cookie, errCookie := c.Request.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, cookie.Value, &session.ID, s.Codecs...)
		if err == nil {
			err = s.load(session)
			if err == nil {
				session.IsNew = false
			}
		}
	}

	return session, err
}

func (s *RedisStore) Save(c *gin.Context, session *sessions.Session) error {
	// Ensure session is not new or has not been modified.
	if session.IsNew || session.Values == nil {
		return nil
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
	if err != nil {
		return err
	}

	// Use the context from Gin which carries the request context
	ctx := c.Request.Context()

	data, err := securecookie.EncodeMulti(session.Name(), session.Values, s.Codecs...)
	if err != nil {
		return err
	}

	// Set the session data in Redis with an expiry (you can customize the expiry duration)
	_, err = s.client.Set(ctx, "session:"+session.ID, data, time.Hour).Result()
	if err != nil {
		return err
	}

	// Set the session cookie using Gin's response writer
	http.SetCookie(c.Writer, sessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

func (s *RedisStore) load(session *sessions.Session) error {
	ctx := context.Background()
	data, err := s.client.Get(ctx, "session:"+session.ID).Result()
	if err != nil {
		return err
	}

	if err = securecookie.DecodeMulti(session.Name(), data, &session.Values, s.Codecs...); err != nil {
		return err
	}

	return nil
}

func switchSessionStore() {
	for {
		ctx := context.Background()
		_, err := rdb.Ping(ctx).Result()

		if err == nil && !redisAvailable.Load() {
			// TODO: fix this later, spending too many cycles on this and its not needed just yet
			//store = NewRedisStore(rdb, []byte(*flag_s_session_store_redis_secret))
			redisAvailable.Store(false)
		} else if err != nil && redisAvailable.Load() {
			// Redis is unavailable but currently in use
			store = cookie.NewStore([]byte(*flag_s_session_store_cookie_secret))
			redisAvailable.Store(false)
		}

		time.Sleep(10 * time.Second) // Check every 10 seconds, adjust as needed
	}
}

func getSessionIncrement(c *gin.Context) {
	session, err := store.Get(c.Request, SESSION_ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	count, ok := session.Values["count"].(int)
	if !ok {
		count = 0
	}
	count++
	session.Values["count"] = count

	if err = session.Save(c.Request, c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func session_start(ctx context.Context) error {
	x509_cert_pool = x509.NewCertPool()
	pem, err := os.ReadFile(*flag_s_session_store_redis_tls_root_ca_path)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}

	if ok := x509_cert_pool.AppendCertsFromPEM(pem); !ok {
		return fmt.Errorf("failed to append CA certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs:            x509_cert_pool,
		InsecureSkipVerify: false, // Set to false to enable verification
	}

	var addresses []string
	if strings.Contains(*flag_s_session_store_redis_servers, ",") { // Replace with actual flag or variable
		addresses = strings.Split(*flag_s_session_store_redis_servers, ",")
	} else {
		addresses = append(addresses, *flag_s_session_store_redis_servers)
	}

	rdb = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:     addresses,
		TLSConfig: tlsConfig,
	})

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		store = cookie.NewStore([]byte(*flag_s_session_store_cookie_secret)) // Update with actual secret
		redisAvailable.Store(false)
		go switchSessionStore() // Start monitoring Redis availability
		return fmt.Errorf("failed to connect to Redis cluster: %v", err)
	}

	// TODO: fix this later
	//store = NewRedisStore(rdb, []byte(*flag_s_session_store_redis_secret)) // Update with actual secret
	redisAvailable.Store(true)

	go switchSessionStore() // Start monitoring Redis availability
	return nil
}
