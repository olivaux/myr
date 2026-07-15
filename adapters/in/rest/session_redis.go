// adapters/in/rest/session_redis.go — store de sessions Redis (scale horizontal)
package rest

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "myr:sess:"

// redisSessionStore implémente sessionBackend avec Redis.
// Permet le partage de sessions entre plusieurs instances myr-api.
type redisSessionStore struct {
	rdb *redis.Client
}

// newRedisSessionStore crée un store Redis à partir d'une URL redis[s]://[:password@]host[:port][/db].
func newRedisSessionStore(url string) (*redisSessionStore, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("URL Redis invalide : %w", err)
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("Redis non disponible : %w", err)
	}
	return &redisSessionStore{rdb: rdb}, nil
}

func (s *redisSessionStore) key(token string) string {
	return redisKeyPrefix + token
}

func (s *redisSessionStore) create(pseudo, role, channel string) (*myrSession, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("génération token impossible : %w", err)
	}
	sess := &myrSession{
		Token:     hex.EncodeToString(b),
		Role:      role,
		Pseudo:    pseudo,
		Channel:   channel,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	if err := s.save(sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *redisSessionStore) save(sess *myrSession) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("session marshal : %w", err)
	}
	ttl := time.Until(sess.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session déjà expirée")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.rdb.Set(ctx, s.key(sess.Token), data, ttl).Err()
}

func (s *redisSessionStore) get(token string) *myrSession {
	if token == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	data, err := s.rdb.Get(ctx, s.key(token)).Bytes()
	if err != nil {
		return nil
	}
	var sess myrSession
	if json.Unmarshal(data, &sess) != nil {
		return nil
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(sess.Token), []byte(token)) != 1 {
		return nil
	}
	return &sess
}

func (s *redisSessionStore) setNetwork(token, networkID string) bool {
	sess := s.get(token)
	if sess == nil {
		return false
	}
	sess.NetworkID = networkID
	return s.save(sess) == nil
}

func (s *redisSessionStore) setChannel(token, channel string) bool {
	sess := s.get(token)
	if sess == nil {
		return false
	}
	sess.Channel = channel
	return s.save(sess) == nil
}

func (s *redisSessionStore) delete(token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.rdb.Del(ctx, s.key(token)) //nolint:errcheck
}

// list retourne jusqu'à 10 000 sessions actives (SCAN + MGET).
func (s *redisSessionStore) list() []*myrSession {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cursor uint64
	var keys []string
	for {
		batch, next, err := s.rdb.Scan(ctx, cursor, redisKeyPrefix+"*", 100).Result()
		if err != nil {
			log.Printf("Redis SCAN error: %v", err)
			break
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 || len(keys) >= 10000 {
			break
		}
	}
	if len(keys) == 0 {
		return nil
	}

	vals, err := s.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil
	}

	now := time.Now()
	out := make([]*myrSession, 0, len(vals))
	for _, v := range vals {
		if v == nil {
			continue
		}
		raw, ok := v.(string)
		if !ok {
			continue
		}
		var sess myrSession
		if json.Unmarshal([]byte(raw), &sess) != nil {
			continue
		}
		if now.Before(sess.ExpiresAt) {
			out = append(out, &sess)
		}
	}
	return out
}
