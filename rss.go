package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/i-bielik/boot-dev-gator/internal/database"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	client := &http.Client{}
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got not OK status code: %d", resp.StatusCode)
	}
	// Unmarshal the XML data into the RSSFeed struct
	var feed RSSFeed
	err = xml.Unmarshal(data, &feed)
	if err != nil {
		return nil, err
	}

	// Unescape HTML entities in title, description
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	// Unescape HTML entities in each item
	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}

	return &feed, nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("expected two arguments: <feed-name> <feed-url>")
	}
	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]

	// return logged in user info from db
	// existingUser, err := s.db.GetUser(context.Background(), s.Config.CurrentUserName)
	// if err != nil {
	// 	if err.Error() == "sql: no rows in result set" {
	// 		return fmt.Errorf("user does not exist: %s", s.Config.CurrentUserName)
	// 	}
	// 	return fmt.Errorf("could not check existing user: %w", err)
	// }

	// Create a new feed entry
	var feed database.CreateFeedParams
	feed.ID = uuid.New()
	feed.CreatedAt = time.Now()
	feed.UpdatedAt = time.Now()
	feed.Name = feedName
	feed.Url = feedURL
	feed.UserID = user.ID

	// Insert the feed into the database
	data, err := s.db.CreateFeed(context.Background(), feed)
	if err != nil {
		return fmt.Errorf("could not add feed: %w", err)
	}
	fmt.Printf("Feed added: %+v\n", data)

	// Add feed follow entry for the current user
	var follow database.CreateFeedFollowParams
	follow.ID = uuid.New()
	follow.CreatedAt = time.Now()
	follow.UpdatedAt = time.Now()
	follow.UserID = user.ID
	follow.FeedID = data.ID
	// Insert the feed follow into the database
	_, err = s.db.CreateFeedFollow(context.Background(), follow)
	if err != nil {
		return fmt.Errorf("could not follow feed after adding: %w", err)
	}
	fmt.Printf("Automatically followed feed: %s\n", feedName)

	return nil
}

func handlerListFeeds(s *state, cmd command) error {
	feeds, err := s.db.ListFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("could not list feeds: %w", err)
	}
	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}
	fmt.Printf("Feeds:\n")
	for _, feed := range feeds {
		fmt.Printf("Name: %s, URL: %s, User: %s\n", feed.Name, feed.Url, feed.UserName.String)
	}
	return nil
}

func handlerFollowFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("expected one argument: <feed-url>")
	}
	feedURL := cmd.Args[0]

	// return logged in user info from db
	// existingUser, err := s.db.GetUser(context.Background(), s.Config.CurrentUserName)
	// if err != nil {
	// 	if err.Error() == "sql: no rows in result set" {
	// 		return fmt.Errorf("user does not exist: %s", s.Config.CurrentUserName)
	// 	}
	// 	return fmt.Errorf("could not check existing user: %w", err)
	// }

	// Fetch the feed to get its ID
	feed, err := s.db.ListFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("could not fetch feed: %w", err)
	}

	// Create a new feed follow entry
	var follow database.CreateFeedFollowParams
	follow.ID = uuid.New()
	follow.CreatedAt = time.Now()
	follow.UpdatedAt = time.Now()
	follow.UserID = user.ID
	follow.FeedID = feed.ID

	// Insert the feed follow into the database
	data, err := s.db.CreateFeedFollow(context.Background(), follow)
	if err != nil {
		return fmt.Errorf("could not follow feed: %w", err)
	}
	fmt.Printf("Feed followed: %+v\n", data)

	return nil
}

func handlerListFeedFollows(s *state, cmd command, user database.User) error {
	// return logged in user info from db
	// existingUser, err := s.db.GetUser(context.Background(), s.Config.CurrentUserName)
	// if err != nil {
	// 	if err.Error() == "sql: no rows in result set" {
	// 		return fmt.Errorf("user does not exist: %s", s.Config.CurrentUserName)
	// 	}
	// 	return fmt.Errorf("could not check existing user: %w", err)
	// }

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("could not list feed follows: %w", err)
	}
	if len(follows) == 0 {
		fmt.Println("No feed follows found.")
		return nil
	}
	fmt.Printf("Feed follows:\n")
	for _, follow := range follows {
		fmt.Printf("Feed name: %s\n", follow.FeedName)
	}
	return nil
}
