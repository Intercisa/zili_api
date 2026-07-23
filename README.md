# Zili App

A baby growth and feeding tracker dashboard.

## Prerequisites

- Docker
- Docker Compose

---

## Option 1: Docker Compose (recommended)

### Start everything

```bash
docker compose up -d
```

### Stop everything

```bash
docker compose down
```

---

## Option 2: Manual Docker setup

### 1. Start PostgreSQL

```bash
docker run -d \
  --name postgresql \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  -v $(pwd)/data:/var/lib/postgresql/data \
  --restart always \
  postgres
```

### 2. Run database migration

```bash
docker cp zili_migration.sql postgresql:/tmp/zili_migration.sql
docker exec -it postgresql psql -U user -d zili -v ON_ERROR_STOP=1 -f /tmp/zili_migration.sql
```

### 3. Build and run the app

```bash
docker build -t zili-app . && docker run -d \
  --name zili-app \
  --network zili-network \
  --env-file .env \
  -p 8081:8081 \
  zili-app
```

---

## Environment variables (.env)

```env
DB_HOST=postgresql
DB_PORT=5432
DB_USER=user
DB_PASSWORD=password
DB_NAME=zili
```

---

## Database

Connect to the database:

```bash
docker exec -it postgresql psql -U user -d zili
```

Run migration manually:

```bash
docker cp zili_migration.sql postgresql:/tmp/zili_migration.sql
docker exec -it postgresql psql -U user -d zili -v ON_ERROR_STOP=1 -f /tmp/zili_migration.sql
```

---

## Access

Once running, open the dashboard at:

```
http://localhost:8081
```

