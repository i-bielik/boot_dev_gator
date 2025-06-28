package main

import (
	"fmt"

	"github.com/i-bielik/boot-dev-gator/internal/config"
)

// state and command structs
type state struct {
	Config *config.Config
}

type command struct {
	Name string
	Args []string
}

type commands struct {
	Data map[string]func(*state, command) error
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("username is required")
	}
	username := cmd.Args[0]
	err := s.Config.SetUser(username)
	if err != nil {
		return fmt.Errorf("could not set user: %w", err)
	}
	fmt.Printf("User set to: %s\n", username)
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	if handler, ok := c.Data[cmd.Name]; ok {
		return handler(s, cmd)
	}
	return fmt.Errorf("unknown command: %s", cmd.Name)
}

func (c *commands) register(name string, f func(*state, command) error) {
	if c.Data == nil {
		c.Data = make(map[string]func(*state, command) error)
	}
	c.Data[name] = f
}
