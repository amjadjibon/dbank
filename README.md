# dbank
Digital Bank - A modern banking application written in Go

## Overview
dbank is a digital banking platform that provides banking services including account management and transactions.

## Prerequisites
- Go 1.24 or higher
- Docker and Docker Compose (for containerized deployment)
- PostgreSQL
- MongoDB
- RabbitMQ
- Redis

## Getting Started

### Run with Docker Compose (Recommended)

The easiest way to run dbank is using Docker Compose:

```bash
# Build and start all services
docker compose up -d

# To rebuild containers after making changes
docker compose up -d --build

# Check logs
docker compose logs -f app

# Stop all services
docker compose down
```

This will start:
- dbank application on port 8080
- PostgreSQL on port 5432
- Redis on port 6379
- RabbitMQ on port 5672 (management UI on 15672)
- MongoDB on port 27017

### Manual Setup

#### 1. Install Dependencies

Make sure you have the following services running:
- PostgreSQL: create a database named `dbank`
- Redis
- RabbitMQ
- MongoDB

#### 2. Build the application

```bash
make build
```

#### 3. Run database migrations

```bash
./bin/dbank migrate
```

#### 4. Start the server

```bash
./bin/dbank serve
```

## Environment Variables

The application uses the following environment variables:

```
ENV=dev           # Environment (dev, prod)
DB_URL=postgres://postgres:postgres@localhost:5432/dbank
REDIS_URL=localhost:6379
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
MONGO_URL=mongodb://localhost:27017/dbank
```

## API Documentation

The API documentation is available at `http://localhost:8080/swagger/` when the server is running.

### Running Tests

```bash
make test
```

### Makefile Commands

```bash
make build          # Build the application
make build-release  # Build for release
make test           # Run tests
make lint           # Run linter
```
