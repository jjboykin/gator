package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jjboykin/gator/internal/config"
	"github.com/jjboykin/gator/internal/database"
	_ "github.com/lib/pq"
)

type state struct {
	db        *database.Queries
	configPtr *config.Config
}
type command struct {
	name        string
	description string
	args        []string
}

type commands struct {
	commandHandler map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.commandHandler[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.commandHandler[cmd.name]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return handler(s, cmd)
}

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

func main() {

	programState := state{}
	cfg, err := config.Read()
	if err != nil {
		fmt.Println("Error reading config:", err)
		os.Exit(1)
	}
	programState.configPtr = &cfg

	db, err := sql.Open("postgres", programState.configPtr.DBUrl)
	if err != nil {
		log.Fatal(errors.New(err.Error()))
		return
	}
	dbQueries := database.New(db)
	programState.db = dbQueries

	// Initialize the commands struct with its map
	cliCommands := commands{
		commandHandler: make(map[string]func(*state, command) error),
	}

	//cliCommands.register("addfeed", handlerAddFeed)
	cliCommands.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cliCommands.register("agg", handlerAggregator)
	cliCommands.register("browse", middlewareLoggedIn(handlerBrowse))
	cliCommands.register("feeds", handlerFeeds)
	cliCommands.register("follow", middlewareLoggedIn(handlerFollow))
	cliCommands.register("following", middlewareLoggedIn(handlerFollowing))
	cliCommands.register("login", handlerLogin)
	cliCommands.register("register", handlerRegister)
	cliCommands.register("reset", handlerReset)
	cliCommands.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cliCommands.register("users", handlerUsers)

	// Check if we have enough arguments
	if len(os.Args) < 2 {
		fmt.Println("Error: not enough arguments")
		os.Exit(1)
	}

	// Create command from arguments
	cmd := command{
		name: os.Args[1],
		args: os.Args[2:],
	}

	// Run the command
	err = cliCommands.run(&programState, cmd)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

}

// Handlers
func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return errors.New("no command args given")
	}

	if len(cmd.args) != 2 {
		return errors.New("incorrect number of command args given")
	}

	name := cmd.args[0]
	url := cmd.args[1]

	feedParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	}
	feed, err := s.db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		fmt.Println("Error:", err)
	}

	feedFollowParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	follow, err := s.db.CreateFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Println("Feed: ", feed.ID, feed.CreatedAt, feed.UpdatedAt, feed.Name, feed.Url, feed.UserID)
	fmt.Println("Follow: ", follow.ID, follow.CreatedAt, follow.UpdatedAt, follow.FeedName, follow.UserName)

	return nil
}

func handlerAggregator(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("time_between_reqs not given")
	}

	if len(cmd.args) > 1 {
		return errors.New("too many command args given")
	}

	time_between_reqs := cmd.args[0]
	timeBetweenRequests, err := time.ParseDuration(time_between_reqs)
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)
	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := 2

	if len(cmd.args) > 0 {
		limitArg, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}
		limit = limitArg
	}

	userParams := database.GetPostsForUserParams{
		ID:    user.ID,
		Limit: int32(limit),
	}

	posts, err := s.db.GetPostsForUser(context.Background(), userParams)
	if err != nil {
		return fmt.Errorf("couldn't get posts for user: %w", err)
	}

	for _, post := range posts {
		fmt.Printf("%s from %s\n", post.PublishedAt, post.FeedName)
		fmt.Printf("--- %s ---\n", post.Title)
		fmt.Printf("    %v\n", post.Description.String)
		fmt.Printf("Link: %s\n", post.Url)
		fmt.Println("==================================================")
	}
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return errors.New("too many command args given")
	}

	feeds, err := s.db.GetFeedsWithUserName(context.Background())
	if err != nil {
		fmt.Println("Error:", err)
	}

	for _, feed := range feeds {
		fmt.Println(feed.Name, feed.Url, feed.UserName)
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return errors.New("no command args given")
	}

	if len(cmd.args) != 1 {
		return errors.New("too many command args given")
	}

	url := cmd.args[0]

	feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		fmt.Println("Error:", err)
	}

	feedFollowParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	follow, err := s.db.CreateFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Println(follow.FeedName)
	fmt.Println(follow.UserName)

	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 0 {
		return errors.New("too many command args given")
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), s.configPtr.CurrentUserName)
	if err != nil {
		fmt.Println("Error:", err)
	}

	for _, follow := range follows {
		fmt.Println(follow.FeedName)
	}

	return nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("no command args given")
	}

	if len(cmd.args) != 1 {
		return errors.New("too many command args given")
	}

	name := cmd.args[0]

	user, err := s.db.GetUserByName(context.Background(), name)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	err = s.configPtr.SetUser(user.Name)
	if err != nil {
		return err
	}

	fmt.Printf("User set to: %s\n", cmd.args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("no command args given")
	}

	if len(cmd.args) != 1 {
		return errors.New("too many command args given")
	}

	// Ensure that a name was passed in the args.
	name := cmd.args[0]

	// Create a new user in the database. It should have access to the CreateUser query through the state -> db struct.
	userParams := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
	}

	user, err := s.db.CreateUser(context.Background(), userParams)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Set the current user in the config to the given name.
	err = s.configPtr.SetUser(user.Name)
	if err != nil {
		return err
	}

	//Print a message that the user was created, and log the user's data to the console for your own debugging.
	fmt.Printf("User created: %s\n", user.Name)
	return nil
}

func handlerReset(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return errors.New("too many command args given")
	}

	err := s.db.DeleteUsers(context.Background())
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	return nil
}

func handlerUsers(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return errors.New("too many command args given")
	}

	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	for _, user := range users {
		output := user.Name
		if s.configPtr.CurrentUserName == user.Name {
			output += " (current)"
		}
		fmt.Println(output)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return errors.New("no command args given")
	}

	if len(cmd.args) != 1 {
		return errors.New("too many command args given")
	}

	url := cmd.args[0]

	feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		fmt.Println("Error:", err)
	}

	feedFollowParams := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}

	err = s.db.DeleteFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		fmt.Println("Error:", err)
	}

	return nil
}

// Middleware
func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUserByName(context.Background(), s.configPtr.CurrentUserName)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
}

// Other functions
func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	feed := &RSSFeed{}
	err = xml.Unmarshal(data, feed)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed from %s: %w", feedURL, err)
	}

	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	return feed, nil
}

func scrapeFeeds(s *state) error {

	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	nullableTimeSQL := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	feedToMarkParams := database.MarkFeedFetchedParams{
		ID:            feed.ID,
		LastFetchedAt: nullableTimeSQL,
		UpdatedAt:     nullableTimeSQL.Time,
	}

	err = s.db.MarkFeedFetched(context.Background(), feedToMarkParams)
	if err != nil {
		return err
	}

	fetchedFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}

	//fmt.Printf("Feed Title: %s\n", fetchedFeed.Channel.Title)
	//fmt.Printf("Feed Description: %s\n", fetchedFeed.Channel.Description)
	//fmt.Println("Feed Items: ")
	for _, item := range fetchedFeed.Channel.Item {

		nullableStringDescription := sql.NullString{
			String: item.Description,
			Valid:  true,
		}

		timeFormats := []string{
			"2006-01-02 15:04:05",
			"2006/01/02",
			"02 Jan 2006",
			"01/02/2006 15:04:05 MST",
			time.RFC3339,
		}

		var parsedPubDate time.Time

		for _, layout := range timeFormats {
			parsedTime, err := time.Parse(layout, item.PubDate)

			if err != nil {
				//fmt.Println("Error parsing date:", err)
				continue
			} else {
				parsedPubDate = parsedTime
				break
			}
		}

		postParams := database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       item.Title,
			Url:         item.Link,
			Description: nullableStringDescription,
			PublishedAt: parsedPubDate,
			FeedID:      feed.ID,
		}

		post, err := s.db.CreatePost(context.Background(), postParams)
		if err != nil {
			fmt.Println("Error:", err)
		}

		fmt.Printf("Post '%s' created as id=%s\n", post.Title, post.ID)

	}

	return nil
}
