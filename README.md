# Materix

## Introduction
### Inspiration
I came across this image and thought, "I could build that!"—so here we are.
<img src="./materix-inspo.jpeg" alt="Project inspiration" width="75%" >

### Key Features
- Find, add, manage and view friends on the app.
- Share your free time with specific friends or make it visible to all your contacts.
- Add tags to your free times for easy filtering.

## Table of Contents
- [Installation](#installation)
- [Usage](#usage)
- [API Endpoints](#api-endpoints)

## Installation
### Prerequisites
- Go 1.x
- PostgreSQL 15+
- [Go migration tool ](https://github.com/golang-migrate/migrate/v4)

### Steps
1. Clone the repo:
   ```bash
   git clone https://github.com/AustinMusiku/Materix-go.git
   cd Materix-go
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```
3. Create an `.envrc` file and configure it with your environment vars shown in `.envrc.example`

4. Run the database migrations:
   ```bash
   migrate -path ./db/migrations -database <DATABASE_URL> up
   ```
5. Build the api:
   ```bash
   go build -o ./bin/api ./cmd/api
   ```

6. Run the api:
   ```bash
   ./bin/api -db-dsn=<DATABASE_URL>
   ```

## Usage
To check available endpoints and interact with the API, visit the [OpenAPI spec doc](./openapi.yaml).

Here's a quick example of how to create a new free time:

```bash
curl -X POST "http://localhost:4000/api/free" \
-H "Authorization: Bearer YOUR_TOKEN" \
-H "accept: application/json" \
-H "Content-Type: application/json" \
-d '{
        "start_time": "2024-09-19T12:00:22.094Z",
        "end_time": "2024-09-19T16:30:22.094Z",
        "tags": [
            "overwatch", "apex", "codm"
        ],
        "visibility": "private",
    }'
```

## API Endpoints

### Authentication
| Method | Endpoint | Require Auth | Description |
|--------|----------|--------------|-------------|
POST | `/auth/signup` | False | Register new user |
POST | `/auth/login` | False | Sign in an existing user via email/pw |
GET | `/auth/callback` | False | Oauth callback handler |

### Users
| Method | Endpoint | Require Auth | Description |
|--------|----------|--------------|-------------|
GET | `/users/{id}` | False | Fetch user's details|
GET | `/users/{search}` | False | Search for a particular user|
GET | `/users/me` | True | Fetch authenticated user's details|
PATCH | `/users/me` | True | Update authenticated user's details|
DELETE | `/users/me` | True | Delete authenticated user's account|

### Friends
| Method | Endpoint | Require Auth | Description |
|--------|----------|--------------|-------------|
GET | `/friends` | True | Fetch authenticated user's friends|
GET | `/friends/search` | True | search among authenticated user's friends|
GET | `/friends/{id}` | True | Remove authenticated user's friend|
GET | `/friends/requests/sent` | True | Fetch authenticated user's outgoing friend requests|
GET | `/friends/requests/received` | True | Fetch authenticated user's incoming friend requests|
POST | `/friends/requests` | True | Send a friend request to a user |
PUT | `/friends/requests/{id}` | True | Accept incoming friend request|
DELETE | `/friends/requests/{id}` | True | Reject incoming friend request/Cancel outgoing request|
GET | `/friends/free` | True | Fetch free time slots for authenticated user's friends |
GET | `/friends/{id}/free` | True | Fetch free time slots for authenticated user's friend |

### Free Time
| Method | Endpoint | Require Auth | Description |
|--------|----------|--------------|-------------|
GET | `/free` | True | Fetch authenticated user's free time slots|
POST | `/free` | True | Publish a new free time slot |
PATCH | `/free/{id}` | True | Modify free time slot details |
DELETE | `/free/{id}` | True | Remove authenticated user's free time slot|

For a full list of API endpoints and parameters, check the [OpenAPI spec](./openapi/openapi.yml).
