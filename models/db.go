package models

import (
	"bytes"
	"fmt"
	"log"

	"github.com/go-redis/redis"
)

// Client holds the redis db connection. It needs to be initialized with
//	models.ConnectRedis()
// first.
var Client *redis.Client

// ConnectRedis initilizes the redis db connection. The connection can be
// accessed via
//	models.Client
// afterwards. This function panics if no connection can be established.
func ConnectRedis(addr string) {
	Client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := Client.Ping().Result()

	if err != nil {
		log.Panic(err)
	}
}

// Article represents an article containing Author and Title.
type Article struct {
	Title, Author string
}

// MarshalBinary an article.
func (a Article) MarshalBinary() ([]byte, error) {
	// A simple encoding: plain text.
	var b bytes.Buffer
	fmt.Fprintln(&b, a.Author, a.Title)
	return b.Bytes(), nil
}

// UnmarshalBinary an article.
func (a *Article) UnmarshalBinary(data []byte) error {
	// A simple encoding: plain text.
	b := bytes.NewBuffer(data)
	_, err := fmt.Fscanln(b, &a.Author, &a.Title)
	return err
}
