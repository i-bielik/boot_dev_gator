package main

import (
	"context"
	"fmt"

	"github.com/i-bielik/boot-dev-gator/internal/database"
)

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		// Check if user is logged in
		if s.Config.CurrentUserName == "" {
			return fmt.Errorf("user not logged in")
		}

		// return logged in user info from db
		user, err := s.db.GetUser(context.Background(), s.Config.CurrentUserName)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				return fmt.Errorf("user does not exist: %s", s.Config.CurrentUserName)
			}
			return fmt.Errorf("could not check existing user: %w", err)
		}

		// Call the handler with the user
		return handler(s, cmd, user)
	}
}
