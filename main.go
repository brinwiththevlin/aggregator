package main

import (
	"aggregator/internal/config"
	"errors"
	"fmt"
	"log"
	"os"
)

type state struct {
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

	handlers := make(map[string]func(*state, command) error)
	cmds := commands{handler: handlers}
	cmds.register("login", handlerLogin)


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
	err := s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("username set to %s\n", cmd.args[0])
	return nil
}
