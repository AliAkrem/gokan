# Gokan — headless real-time chat application built with Go

> Drop real-time 1-1 messaging into any backend in minutes.  
> Built with Go. No lock-in. No UI. Just a clean REST + WebSocket API you control.

[Screencast from 2026-05-01 23-41-28.webm](https://github.com/user-attachments/assets/99796ac5-32d5-46f2-bbc9-6d6859d90fca)

---

## Table of Contents

- [Why Gokan](#why-gokan)
- [Features](#features)
- [Architecture](#architecture)
- [Quickstart](#quickstart)
- [API Reference](#api-reference)
- [Frontend Example](#frontend-example)
- [Configuration Reference](#configuration-reference)
- [Support](#support)

---

## Why Gokan

Most chat services lock you into their infrastructure, their pricing, and their data model. Gokan is different — it's headless, meaning it has no UI and no opinion about your app. It syncs users from your existing auth system, manages rooms and messages, and exposes a clean REST + WebSocket API you control.

---

## Features

- **Real-time messaging** — WebSocket-based delivery with Redis Pub/Sub fan-out
- **Offline message queuing** — messages queued in Redis and delivered on reconnect
- **Secure WebSocket tickets** — short-lived Redis-backed tickets prevent unauthorized WS upgrades
- **User sync** — automatically syncs users from your backend via `USER_INFO_URL`, no manual user management
- **Room management** — create, update, delete rooms with participant tracking
- **JWT / JWKS auth** — works with any identity provider

---

## Architecture

```
Your App
   │
   ├── REST API ─────────────────── MongoDB 7
   │    ├── /api/v1/users                  └── users, rooms, messages
   │    ├── /api/v1/rooms
   │    ├── /api/v1/messages
   │    └── /api/v1/tickets
   │
   └── WebSocket (/ws) ─────────────────── Redis 7
        └── Hub (Pub/Sub)                  ├── WS tickets (TTL)
             └── Rooms                     └── offline queues (TTL)
```

---

## Quickstart

### 1. Clone the Repository

```bash
git clone https://github.com/aliakrem/gokan.git
cd gokan
```

### 2. Configure Environment Variables

```bash
cp .env.example .env
```

At minimum, set:

```env
MONGO_URI=mongodb://localhost:27017
JWT_SECRET=your-secret        # or use JWKS_URL instead
USER_INFO_URL=https://your-api.com/me
```

### 3. Start Dependencies

```bash
docker-compose up -d
```

Starts MongoDB on `:27017` and Redis on `:6379`.

### 4. Run the Server

```bash
go run main.go
```

---

## API Reference

### Health & Metrics

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check endpoint with service status |
| `GET` | `/metrics` | Application metrics including WebSocket connections |

### WebSocket

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/ws` | WebSocket connection endpoint (requires valid ticket) |

### API v1

> All routes require JWT authentication.

#### Tickets

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/tickets` | Generate WebSocket connection ticket |

#### Users

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/users/:id` | Get user by ID |
| `GET` | `/api/v1/users` | List users |

#### Rooms

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/rooms` | Create a new room |
| `GET` | `/api/v1/rooms/:id` | Get room details |
| `GET` | `/api/v1/rooms` | List rooms |
| `PUT` | `/api/v1/rooms/:id` | Update room |
| `DELETE` | `/api/v1/rooms/:id` | Delete room |

#### Messages

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/messages` | Send a message |
| `GET` | `/api/v1/messages` | Get messages (with room filtering) |
| `GET` | `/api/v1/messages/:id` | Get message by ID |

---

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

---

## Configuration Reference

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8080` | No |
| `MONGO_URI` | MongoDB connection string | — | Yes |
| `MONGO_DB` | MongoDB database name | `gokan` | No |
| `JWT_SECRET` | JWT secret key | — | Yes\* |
| `JWKS_URL` | JWKS endpoint URL | — | Yes\* |
| `JWT_CLAIM_KEY` | JWT claim key for user ID | `sub` | No |
| `USER_INFO_URL` | User info endpoint | — | Yes |
| `USER_SYNC_TTL_SECONDS` | User sync cache TTL | `300` | No |
| `PING_INTERVAL_SEC` | WebSocket ping interval | `30` | No |
| `PONG_TIMEOUT_SEC` | WebSocket pong timeout | `10` | No |
| `MAX_MESSAGE_BYTES` | Max WebSocket message size | `65536` | No |
| `WS_TICKET_TTL_SEC` | WebSocket ticket TTL | `30` | No |
| `OFFLINE_QUEUE_TTL_DAYS` | Offline message queue TTL | `30` | No |
| `REDIS_HOST` | Redis host | `localhost` | No |
| `REDIS_PORT` | Redis port | `6379` | No |
| `REDIS_PASSWORD` | Redis password | — | No |
| `REDIS_DB` | Redis database number | `0` | No |
| `REDIS_USE_TLS` | Enable Redis TLS | `false` | No |
| `ALLOWED_ORIGINS` | CORS allowed origins | `*` | No |
| `LOG_LEVEL` | Logging level | `info` | No |

\* Either `JWT_SECRET` or `JWKS_URL` is required.

---

## Support

For issues and questions, please [open an issue on GitHub](https://github.com/aliakrem/gokan/issues).
