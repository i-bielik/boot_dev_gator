package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
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

func scrapeFeeds(s *state) error {
	ctx := context.Background()
	// Fetch next feed to scrape
	feed, err := s.db.GetNextFeedToFetch(ctx)
	if err != nil {
		return fmt.Errorf("could not get next feed to fetch: %w", err)
	}
	// Update the feed's last fetched time
	err = s.db.UpdateFetchedFeed(ctx, feed.ID)
	if err != nil {
		return fmt.Errorf("could not update fetched feed: %w", err)
	}
	// Fetch the feed
	fetchedFeed, err := fetchFeed(ctx, feed.Url)
	if err != nil {
		return fmt.Errorf("could not fetch feed: %w", err)
	}
	// fmt.Println("Fetched feed: ", feed.Name)
	// Iterate over each item in the feed
	for _, item := range fetchedFeed.Channel.Item {
		publishedAt := sql.NullTime{}
		if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
			publishedAt = sql.NullTime{
				Time:  t,
				Valid: true,
			}
		}

		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			FeedID:    feed.ID,
			Title:     item.Title,
			Description: sql.NullString{
				String: item.Description,
				Valid:  true,
			},
			Url:         item.Link,
			PublishedAt: publishedAt,
		})
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue
			}
			log.Printf("Couldn't create post: %v", err)
			continue
		}
	}
	fmt.Printf("Feed %s collected, %v posts found\n", feed.Name, len(fetchedFeed.Channel.Item))
	return nil
}

func handlerBrowsePosts(s *state, cmd command, user database.User) error {
	limit := 2
	if len(cmd.Args) == 1 {
		if specifiedLimit, err := strconv.Atoi(cmd.Args[0]); err == nil {
			limit = specifiedLimit
		} else {
			return fmt.Errorf("invalid limit: %w", err)
		}
	}

	posts, err := s.db.GetPostPerUser(context.Background(), database.GetPostPerUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("couldn't get posts for user: %w", err)
	}

	fmt.Printf("Found %d posts for user %s:\n", len(posts), user.Name)
	for _, post := range posts {
		fmt.Printf("%s --- %s --- %s\n", post.PublishedAt.Time.Format("2021-01-01"), post.FeedName, post.Title)
		fmt.Println("=====================================")
	}

	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("expected two arguments: <feed-name> <feed-url>")
	}
	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]

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

func handlerUnfollowFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("expected one argument: <feed-url>")
	}
	feedURL := cmd.Args[0]

	deleteInfo := database.DeleteFeedFollowParams{
		UserID: user.ID,
		Url:    feedURL,
	}

	// Delete the feed follow entry for the current user
	err := s.db.DeleteFeedFollow(context.Background(), deleteInfo)
	if err != nil {
		return fmt.Errorf("could not unfollow feed: %w", err)
	}
	fmt.Printf("Unfollowed feed: %s\n", feedURL)

	return nil
}
