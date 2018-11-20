package main

import (
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"strconv"

	"github.com/s-gheldd/article-redis/models"
	"github.com/urfave/cli"

	"github.com/go-redis/redis"
)

const (
	keyArticles = "articles"
	keyRatings  = "ratings:"
	keyScores   = "scores"
)

func main() {
	models.ConnectRedis("localhost:6379")

	app := setUpCli()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func bestArticles(n int64) ([]redis.Z, error) {
	return models.Client.ZRevRangeWithScores(keyScores, 0, n).Result()
}

func rateArticle(key, account string, rating float64) error {
	if rating > 5 || rating < 0 {
		return fmt.Errorf("rating must be between 0 and 5, got %.2f", rating)
	}

	err := models.Client.Watch(func(tx *redis.Tx) error {
		add, err := tx.SAdd(keyRatings+key, account).Result()
		if err != nil {
			return err
		}
		if add == 0 {
			return fmt.Errorf("account:%s already rated article:%s", account, key)
		}

		card, err := tx.SCard(keyRatings + key).Result()
		if err != nil {
			return err
		}

		score, err := tx.ZScore(keyScores, key).Result()
		if err != nil {

			if err == redis.Nil {
				score = 0
			} else {
				return err
			}
		}

		score = score * float64(card-1)
		score += rating

		return tx.ZAdd(keyScores, redis.Z{Member: key, Score: score / float64(card)}).Err()

	}, keyRatings+key, keyScores)

	return err
}

func insertArticle(title, author string) (string, error) {
	article := models.Article{Title: title,
		Author: author}

	key := key(title, author)

	_, err := models.Client.HSet(keyArticles, key, article).Result()
	if err != nil {
		return "", err
	}

	return key, nil
}

func getArtricle(key string) (*models.Article, error) {
	res := models.Client.HGet(keyArticles, key)

	if res.Err() != nil {
		return nil, res.Err()
	}
	article := &models.Article{}
	res.Scan(article)
	return article, nil
}

func key(title, author string) string {

	hash := fnv.New32a()
	start := []rune(author)[0]

	fmt.Fprintf(hash, "%s%s", title, author)
	sum := hash.Sum32() >> 22

	return fmt.Sprintf("%c%d", start, sum)
}

func setUpCli() *cli.App {
	app := cli.NewApp()
	app.Name = "article-redis"
	app.Usage = "article redis frontend"

	app.Commands = []cli.Command{
		{
			Name:  "add",
			Usage: "add title author",
			Action: func(c *cli.Context) error {

				if c.NArg() != 2 {
					return fmt.Errorf("add needs two arguments: title author")
				}
				key, err := insertArticle(c.Args().Get(0), c.Args().Get(1))
				if err != nil {
					return err
				}
				fmt.Println(key)
				return nil
			},
		},
		{
			Name:  "show",
			Usage: "show key",
			Action: func(c *cli.Context) error {
				if c.NArg() != 1 {
					return fmt.Errorf("show needs one argument: key")
				}
				art, err := getArtricle(c.Args().Get(0))
				if err != nil {
					return err
				}
				fmt.Printf("%+v\n", *art)
				return nil
			},
		},
		{
			Name:  "rate",
			Usage: "rate key user rating",
			Action: func(c *cli.Context) error {
				if c.NArg() != 3 {
					return fmt.Errorf("rate needs three arguments: key user action")
				}
				rating, err := strconv.ParseFloat(c.Args().Get(2), 64)
				if err != nil {
					return err
				}
				err = rateArticle(c.Args().Get(0), c.Args().Get(1), rating)

				return err
			},
		},
		{
			Name:  "best",
			Usage: "best n",
			Action: func(c *cli.Context) error {
				if c.NArg() != 1 {
					return fmt.Errorf("best needs one argument: best n")
				}
				n, err := strconv.ParseInt(c.Args().Get(0), 10, 64)
				if err != nil {
					return err
				}
				arts, err := bestArticles(n)
				if err != nil {
					return err
				}

				for _, art := range arts {
					fmt.Printf("%s %.2f\n", art.Member, art.Score)
				}
				return nil
			},
		},
		{
			Name:  "fill",
			Usage: "fill",
			Action: func(c *cli.Context) error {
				fmt.Println(insertArticle("ETH", "Metcalfe/Boggs"))
				fmt.Println(insertArticle("ERM", "Chen"))
				fmt.Println(insertArticle("GoTo", "Dijkstra"))
				fmt.Println(insertArticle("UNIX", "Richie/Thompson"))
				fmt.Println(insertArticle("CSP", "Hoare"))
				fmt.Println(insertArticle("OSI", "Zimmerman"))
				fmt.Println(insertArticle("RDB", "Codd"))
				fmt.Println(insertArticle("CRY", "Diffie/Hellman"))

				return nil
			},
		},
	}
	return app
}
