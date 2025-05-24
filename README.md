# gator
- Gator requires Go (golang) and Postgres to be be installed to run the program.
- The **gator** CLI can be installed using *go install.*
- Gator is configured with a JSON file found at ~/.gatorconfig.json with the following parameters:
    - "db_url":"postgres://postgres:postgres@localhost:5432/gator?sslmode=disable"
    - "current_user_name":"<user_name>"

## Gator commands:
    - addfeed [authenticated]: add a new feed to your user list
    - agg [args: <timeBetweenRequests>]: polls users feeds at the specified interval and scrapes for posts
	- browse [authenticated]: lists all catalogued posts from user feeds
	- feeds: list all feeds
	- follow [authenticated; args: <feed_url>]: follow another user's feed
	- following [authenticated]: list all your user's followed feeds
	- login [args: <user_name>]: login to your user
	- register [args: <user_name>]: create a new user account
	- reset: reset the user and feed lists
	- unfollow [authenticated; args: <feed_url>]: stops following another user's feed
	- users: list all users

## Extending the Project
- Add sorting and filtering options to the browse command
- Add pagination to the browse command
- Add concurrency to the agg command so that it can fetch more frequently
- Add a search command that allows for fuzzy searching of posts
- Add bookmarking or liking posts
- Add a TUI that allows you to select a post in the terminal and view it in a more readable format (either in the terminal or open in a browser)
- Add an HTTP API (and authentication/authorization) that allows other users to interact with the service remotely
- Write a service manager that keeps the agg command running in the background and restarts it if it crashes