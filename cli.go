package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/i-bielik/boot-dev-gator/internal/config"
	"github.com/i-bielik/boot-dev-gator/internal/database"
)

// state and command structs
type state struct {
	db     *database.Queries
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

	// Check if user exists
	existingUser, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("user does not exist: %s", username)
		}
		return fmt.Errorf("could not check existing user: %w", err)
	}
	if existingUser.Name != username {
		return fmt.Errorf("user does not match: %s", username)
	}

	err = s.Config.SetUser(username)
	if err != nil {
		return fmt.Errorf("could not set user: %w", err)
	}
	fmt.Printf("User set to: %s\n", username)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("username is required")
	}
	username := cmd.Args[0]

	var user database.CreateUserParams
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.Name = username

	// Check if user already exists
	existingUser, err := s.db.GetUser(context.Background(), username)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return fmt.Errorf("could not check existing user: %w", err)
	}
	if existingUser.Name == username {
		return fmt.Errorf("user already exists: %s", username)
	}

	data, err := s.db.CreateUser(context.Background(), user)
	if err != nil {
		return fmt.Errorf("could not register user: %w", err)
	}
	fmt.Printf("User registered: %+v\n", data)

	// set user in config
	err = s.Config.SetUser(data.Name)
	if err != nil {
		return fmt.Errorf("could not set user in config: %w", err)
	}

	return nil
}

func handlerReset(s *state, cmd command) error {
	// Reset the users table
	err := s.db.DeleteUsers(context.Background())
	if err != nil {
		return fmt.Errorf("could not reset users: %w", err)
	}
	fmt.Println("Users table reset successfully")
	return nil
}

func handlerListUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("could not list users: %w", err)
	}

	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}

	fmt.Println("Users:")
	for _, user := range users {
		if user.Name == s.Config.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
			continue
		}
		fmt.Printf("* %s\n", user.Name)
	}
	return nil
}

func handlerRssAggregate(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("expected one argument: <time-duration>")
	}
	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("could not parse time duration: %w", err)
	}

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}

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
