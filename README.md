## gator
- Gator requires Go (golang) and Postgres to be be installed to run the program.
- The **gator** CLI can be installed using *go install.*
- Gator is configured with a JSON file found at ~/.gatorconfig.json with the following parameters:
    - "db_url":"postgres://postgres:postgres@localhost:5432/gator?sslmode=disable"
    - "current_user_name":"<user_name>"

# Gator commands:
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
