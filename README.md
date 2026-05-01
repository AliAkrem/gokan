# Gokan

A headless real-time chat application built with Go, designed to manage message synchronization, user management, and chat rooms through a WebSocket-based architecture.

## Overview

Gokan is a backend chat service that provides real-time messaging capabilities through WebSockets, with a REST API for managing users, rooms, and messages. It's designed to be headless, meaning it can be integrated with any application.

## Features

- **Real-time Messaging**: WebSocket-based communication for instant message delivery
- **User Management**: Automatic user synchronization with JWT authentication
- **Room Management**: Create and manage chat rooms with participant tracking
- **Message Persistence**: MongoDB-based message storage with offline message queuing
- **Redis Pub/Sub**: Message queuing and real-time distribution using Redis publish/subscribe
- **WebSocket Tickets**: Secure WebSocket connection tickets with Redis-based TTL
- **Graceful Shutdown**: Proper cleanup of connections and resources
- **Health Monitoring**: Built-in health check and metrics endpoints
- **CORS Support**: Configurable cross-origin resource sharing

## Architecture

- **Backend**: Go 1.25+ with Gin web framework
- **Database**: MongoDB 7 for persistent storage
- **Message Queue**: Redis 7 Pub/Sub for real-time message distribution and offline queuing
- **Cache**: Redis for WebSocket tickets and user session management
- **Authentication**: JWT-based with support for JWKS
- **WebSocket**: Real-time bidirectional communication

## Prerequisites

- Go 1.25 or higher
- Docker and Docker Compose (for MongoDB and Redis)
- Node.js 18+ and npm/yarn (for frontend example)

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/aliakrem/gokan.git
cd gokan
```

### 2. Set Up Environment Variables

Copy the example environment file and configure it:

```bash
cp .env.example .env
```


### 3. Start Dependencies

Start MongoDB and Redis using Docker Compose:

```bash
docker-compose up -d
```

This will start:
- MongoDB on `localhost:27017`
- Redis on `localhost:6379`

### 4. Install Go Dependencies

```bash
go mod download
```

### 5. Run the Application

```bash
go run main.go
```

The server will start on the port specified in your `.env` file (default: 8080).

## API Endpoints

### Health & Metrics

- `GET /health` - Health check endpoint with service status
- `GET /metrics` - Application metrics including WebSocket connections

### WebSocket

- `GET /ws` - WebSocket connection endpoint (requires valid ticket)

### API v1 (All routes require JWT authentication)

#### Tickets
- `POST /api/v1/tickets` - Generate WebSocket connection ticket

#### Users
- `GET /api/v1/users/:id` - Get user by ID
- `GET /api/v1/users` - List users

#### Rooms
- `POST /api/v1/rooms` - Create a new room
- `GET /api/v1/rooms/:id` - Get room details
- `GET /api/v1/rooms` - List rooms
- `PUT /api/v1/rooms/:id` - Update room
- `DELETE /api/v1/rooms/:id` - Delete room

#### Messages
- `POST /api/v1/messages` - Send a message
- `GET /api/v1/messages` - Get messages (with room filtering)
- `GET /api/v1/messages/:id` - Get message by ID

## Frontend Example

A React + TypeScript frontend example is included in `apps/front-end`.

### Setup Frontend

1. Navigate to the frontend directory:

```bash
cd apps/front-end
```

2. Install dependencies:

```bash
npm install
# or
yarn install
```

3. Configure environment variables:

```bash
cp .env.example .env
```

4. Run the development server:

```bash
npm run dev
# or
yarn dev
```


## Configuration Reference

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8080` | No |
| `MONGO_URI` | MongoDB connection string | - | Yes |
| `MONGO_DB` | MongoDB database name | `gokan` | No |
| `JWT_SECRET` | JWT secret key | - | Yes* |
| `JWKS_URL` | JWKS endpoint URL | - | Yes* |
| `JWT_CLAIM_KEY` | JWT claim key for user ID | `sub` | No |
| `USER_INFO_URL` | User info endpoint | - | Yes |
| `USER_SYNC_TTL_SECONDS` | User sync cache TTL | `300` | No |
| `PING_INTERVAL_SEC` | WebSocket ping interval | `30` | No |
| `PONG_TIMEOUT_SEC` | WebSocket pong timeout | `10` | No |
| `MAX_MESSAGE_BYTES` | Max WebSocket message size | `65536` | No |
| `WS_TICKET_TTL_SEC` | WebSocket ticket TTL | `30` | No |
| `OFFLINE_QUEUE_TTL_DAYS` | Offline message queue TTL | `30` | No |
| `REDIS_HOST` | Redis host | `localhost` | No |
| `REDIS_PORT` | Redis port | `6379` | No |
| `REDIS_PASSWORD` | Redis password | - | No |
| `REDIS_DB` | Redis database number | `0` | No |
| `REDIS_USE_TLS` | Enable Redis TLS | `false` | No |
| `ALLOWED_ORIGINS` | CORS allowed origins | `*` | No |
| `LOG_LEVEL` | Logging level | `info` | No |

\* Either `JWT_SECRET` or `JWKS_URL` is required


### WebSocket Connection Issues

1. Ensure you have a valid JWT token
2. Generate a WebSocket ticket via `POST /api/v1/tickets`
3. Use the ticket to connect to `/ws?ticket=<your_ticket>`


## Support

For issues and questions, please open an issue on GitHub.
