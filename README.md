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

## Backup
```bash
#!/bin/bash

set -euo pipefail

CONTAINER_NAME="postgresql"
DB_USER="user"
DB_NAME="zili"

BACKUP_DIR="/home/intercisa/backups/postgresql"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

BACKUP_FILE="$BACKUP_DIR/zili_full_${TIMESTAMP}.sql"
LATEST_FILE="$BACKUP_DIR/zili_full_latest.sql"
LOG_FILE="$BACKUP_DIR/backup_zili.log"

mkdir -p "$BACKUP_DIR"

echo "========================================" | tee -a "$LOG_FILE"
echo "Starting backup: $(date)" | tee -a "$LOG_FILE"
echo "Container: $CONTAINER_NAME" | tee -a "$LOG_FILE"
echo "Database:  $DB_NAME" | tee -a "$LOG_FILE"
echo "Output:    $BACKUP_FILE" | tee -a "$LOG_FILE"

docker exec "$CONTAINER_NAME" pg_dump \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --clean \
    --if-exists \
    --create \
    --inserts \
    --column-inserts \
    > "$BACKUP_FILE"

if [ ! -s "$BACKUP_FILE" ]; then
    echo "ERROR: Backup file was not created or is empty." | tee -a "$LOG_FILE"
    exit 1
fi

cp "$BACKUP_FILE" "$LATEST_FILE"

echo "Backup created successfully:" | tee -a "$LOG_FILE"
ls -lh "$BACKUP_FILE" | tee -a "$LOG_FILE"

echo "Latest backup updated:" | tee -a "$LOG_FILE"
ls -lh "$LATEST_FILE" | tee -a "$LOG_FILE"

echo "Deleting backups older than 30 days..." | tee -a "$LOG_FILE"
find "$BACKUP_DIR" -type f -name "zili_full_*.sql" -mtime +30 -delete

echo "Backup finished: $(date)" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"
```

```bash
crontab -e
```

```bash
0 2 * * * /home/intercisa/backup_zili_full.sh
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

### Restore with backup
```bash
docker exec -i postgresql psql -U user -d postgres < /home/intercisa/backups/postgresql/zili_full_latest.sql
```


---

## Access

Once running, open the dashboard at:

```
http://localhost:8081
```

