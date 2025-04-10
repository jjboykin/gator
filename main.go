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

	// Register the login handler
	cliCommands.register("agg", handlerAggregator)
	cliCommands.register("login", handlerLogin)
	cliCommands.register("register", handlerRegister)
	cliCommands.register("reset", handlerReset)
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

	fmt.Printf("DB URL: %v\n", cfg.DBUrl)
	fmt.Printf("Username: %v\n", cfg.CurrentUserName)

}

func handlerAggregator(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return errors.New("too many command args given")
	}

	feedURL := "https://www.wagslane.dev/index.xml"

	feed, err := fetchFeed(context.Background(), feedURL)
	if err != nil {
		return err
	}
	fmt.Println(*feed)

	return nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("no command args given")
	}

	if len(cmd.args) != 1 {
		return errors.New("too many command args given")
	}

	// Ensure that a name was passed in the args.
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
