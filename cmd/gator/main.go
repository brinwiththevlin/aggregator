package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/brinwiththevlin/aggregator/internal/config"
	"github.com/brinwiththevlin/aggregator/internal/database"
	"github.com/brinwiththevlin/aggregator/internal/rss"
	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type handler struct {
	handler     func(*state, command) error
	description string
}

type commands struct {
	handler map[string]handler
}

func (c *commands) register(name string, f func(*state, command) error, d string) {
	c.handler[name] = handler{handler: f, description: d}
}

func (c *commands) describe() error {
	for cmd, h := range c.handler {
		fmt.Printf("* name: %s, usage: %s\n", cmd, h.description)
	}
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	name := cmd.name
	if name == "help" {
		return c.describe()
	}
	return c.handler[name].handler(s, cmd)

}

func main() {

	args := os.Args
	if len(args) < 2 {
		fmt.Println("Error: Not enough arguments provided")
		os.Exit(1)
	}

	s := state{}
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}
	s.cfg = &cfg

	db, err := sql.Open("postgres", s.cfg.Url)

	if err != nil {
		log.Fatal("could not open postgres database")
	}
	dbQueries := database.New(db)
	s.db = dbQueries

	handlers := make(map[string]handler)
	cmds := commands{handler: handlers}
	cmds.register("login", handlerLogin, "gator login <user_name>")
	cmds.register("register", handlerRegister, "gator register <user_name>")
	cmds.register("reset", handlerReset, "gator reset")
	cmds.register("users", handlerUsers, "gator users")
	cmds.register("agg", handlerAgg, "gator agg <duration>")
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed), "gator addfeed <feed> <url>")
	cmds.register("feeds", handlerFeeds, "gator feeds")
	cmds.register("follow", middlewareLoggedIn(handlerFollow), "gator follow <url>")
	cmds.register("following", middlewareLoggedIn(handlerFollowing), "gator following")
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow), "gator unfollow <url>")
	cmds.register("browse", middlewareLoggedIn(handlerBrowse), "gator browse <limit>")

	commandName := args[1]
	commandArgs := args[2:]

	cmd := command{name: commandName, args: commandArgs}
	err = cmds.run(&s, cmd)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return errors.New("usage: gator login <username>")
	}
	name := cmd.args[0]

	_, err := s.db.GetUser(context.Background(), name)
	if err != nil {
		fmt.Printf("%s is not a registered user\n", name)
		return err
	}

	err = s.cfg.SetUser(name)
	if err != nil {
		return err
	}

	fmt.Printf("username set to %s\n", cmd.args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return errors.New("usage: gator register <username>")
	}

	name := cmd.args[0]
	_, err := s.db.GetUser(context.Background(), name)
	if err == nil {
		fmt.Println("user already exists")
		return errors.New("user already exists")
	}

	arg := database.CreateUserParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: name}
	user, err := s.db.CreateUser(context.Background(), arg)
	if err != nil {
		return err
	}
	fmt.Printf("User created: %+v\n", user)

	err = s.cfg.SetUser(name)
	if err != nil {
		return err
	}
	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.Reset(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("All users have been removed")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}
	for _, u := range users {
		str := "* " + u.Name
		if u.Name == s.cfg.Username {
			str += " (current)"
		}
		fmt.Println(str)
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return errors.New("usage: gator agg <time_between_reqs>")
	}

	delta, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %s", cmd.args[0])
	ticker := time.NewTicker(delta)

	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}

}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return errors.New("usage: gator addfeed <feed name> <URL>")
	}
	arg := database.CreateFeedParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: cmd.args[0], Url: cmd.args[1], UserID: user.ID}
	feed, err := s.db.CreateFeed(context.Background(), arg)
	if err != nil {
		return err
	}
	fmt.Println(feed)
	follow_arg := database.CreateFeedFollowParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), UserID: user.ID, FeedID: feed.ID}
	_, err = s.db.CreateFeedFollow(context.Background(), follow_arg)
	if err != nil {
		return err
	}
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}
	for _, f := range feeds {
		fmt.Println(f)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return errors.New("usage: gator follow <url>")
	}
	url := cmd.args[0]
	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return err
	}
	params := database.CreateFeedFollowParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), FeedID: feed.ID, UserID: user.ID}
	follow, err := s.db.CreateFeedFollow(context.Background(), params)
	if err != nil {
		return err
	}
	fmt.Printf("feed name: %s\n", follow.FeedName)
	fmt.Printf("user: %s\n", follow.UserName)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {

	follows, err := s.db.GetFeedFollowForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}
	for _, f := range follows {
		fmt.Println(f.FeedName)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return errors.New("usage: gator unfollow <feedURL>")
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}

	args := database.DeleteFeedFollowParams{FeedID: feed.ID, UserID: user.ID}
	err = s.db.DeleteFeedFollow(context.Background(), args)
	if err != nil {
		return err
	}
	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	var limit int
	if len(cmd.args) != 1 {
		limit = 2
	} else {
		var err error
		limit, err = strconv.Atoi(cmd.args[0])
		if err != nil {
			return errors.New("limit must be an integer, usage: gator browse <limit>")
		}
	}

	args := database.GetPostsForUserParams{ID: user.ID, Limit: int32(limit)}
	posts, err := s.db.GetPostsForUser(context.Background(), args)
	if err != nil {
		return err
	}
	for _, p := range posts {
		fmt.Println("---")
		fmt.Println(p.Title)
		fmt.Println(p.Url)
		fmt.Println(p.PublishedAt.Time)
		fmt.Println(p.Description.String)
	}
	return nil

}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.Username)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)

	}
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}
	mark_args := database.MarkFeedFetchedParams{ID: feed.ID, UpdatedAt: time.Now()}
	err = s.db.MarkFeedFetched(context.Background(), mark_args)
	if err != nil {
		return err
	}
	rssFeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}

	for _, i := range rssFeed.Channel.Item {
		var desc sql.NullString
		if i.Description != nil && *i.Description != "" {
			desc.String = *i.Description
			desc.Valid = true
		} else {
			desc.Valid = false
		}

		var pub sql.NullTime
		if i.PubDate != nil && *i.PubDate != "" {
			parsedTime, err := parseDate(*i.PubDate)
			if err != nil {
				pub.Valid = false
			} else {
				pub.Time = parsedTime
				pub.Valid = true
			}
		}

		args := database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       i.Title,
			Url:         i.Link,
			Description: desc,
			PublishedAt: pub,
			FeedID:      feed.ID,
		}
		_, err := s.db.CreatePost(context.Background(), args)
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				continue
			}
			return err
		}
		fmt.Println(i.Title)
	}

	return nil
}

func parseDate(s string) (time.Time, error) {
	var timeFormats = []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		"2006-01-02",
		"02 Jan 2006",
	}

	for _, format := range timeFormats {
		ptime, err := time.Parse(format, s)
		if err != nil {
			continue
		}
		return ptime, nil
	}
	return time.Time{}, errors.New("unfamiliar time format")

}
