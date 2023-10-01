package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var rdb *redis.Client

func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:	  "localhost:6379",
		Password: "",
		DB:		  0,
	})
}

func rateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIP := r.RemoteAddr
		currTimestamp := time.Now().Unix()
		key := fmt.Sprintf("rate:limiter:%s:%d", userIP, currTimestamp)

		val, err := rdb.Get(ctx, key).Result()
		if err != nil && err != redis.Nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		requestCount, _ := strconv.Atoi(val)
		if requestCount >= 5 {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		// 1 second expiration
		_, err = rdb.SetNX(ctx, key, 1, time.Second).Result()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if val != "" {
			rdb.Incr(ctx, key)
		}

		next.ServeHTTP(w, r)
	})
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Successful request")
}

func main() {
	http.Handle("/", rateLimiter(http.HandlerFunc(mainHandler)))
	http.ListenAndServe(":8080", nil)
}