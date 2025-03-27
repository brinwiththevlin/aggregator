# Gator

[Brief description of what Gator does]

## Prerequisites

- Go (version 1.16 or higher)
- PostgreSQL (version 9.6 or higher)

## Installation

There are two ways to install Gator:

### Option 1: Clone and Build

```bash
# Clone the repository
git clone https://github.com/brinwiththevlin/aggregator.git

# Navigate to the project directory
cd aggregator

# Build and install
go install .
```
### Option 2: Direct Installation
```bash
go install github.com/brinwiththevlin/aggregator@latest
```

## Configuration
Before running Gator, you need to configure it properly. Create a `.gatorconfig.json` file in your home directotry

```json
{
    "db_url" : "postgres://<user_name>:<password>@localhost:5432/gator?sslmode=disable"
    "current_user_name": <user_name>
}
```
## Usage
Once installed and configured, you can start using Gator with the following commands:

- `gator register <user_name>`: register a new user for your database
- `gator login <user_name>`: login as a registered user
- `gator reset`: remove all users and feed\_follows
- `gator users`: list all users
- `gator feeds`: list all feeds
- `gator following`: list all feeds followed by the currently logged in user
- `gator unfollow <url>`: cause the user to unfollow a feed
- `gator browse <limit>`: quick look at posts on the feeds you follow
- `gator agg <duration>`: continuous fetching of feeds in the database with a wait time of duration
- `gator addfeed <feed> <url>`: add a feed to the feeds table, auto follow for the current user
- `gator follow <url>`: follow the feed for current user

