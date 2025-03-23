package main

import (
	"aggregator/internal/config"
	"aggregator/internal/database"
	"aggregator/internal/rss"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
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

type commands struct {
	handler map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.handler[name] = f
}
func (c *commands) run(s *state, cmd command) error {
	name := cmd.name
	return c.handler[name](s, cmd)

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

	handlers := make(map[string]func(*state, command) error)
	cmds := commands{handler: handlers}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))

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
	if len(cmd.args) == 0 {
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
	if len(cmd.args) == 0 {
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
	feed, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	fmt.Println(feed)
	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
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
	if len(cmd.args) < 1 {
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

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.Username)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)

	}
}
