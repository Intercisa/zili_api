# Zili App

A baby growth and feeding tracker dashboard.

## Prerequisites

- Docker
- Docker Compose

---

## Project structure

```
project/
├── docker-compose.yml
├── create_tables.sql
├── provisioning/
│   ├── datasources/
│   │   └── datasource.yaml
│   └── dashboards/
│       └── dashboard.yaml
├── dashboards/          ← exported Grafana dashboard JSON files
├── grafana-data/        ← Grafana internal state (gitignored)
└── .env
```

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

### Restore database from backup (only when needed)

```bash
docker compose --profile restore up postgresql-restore
```

This uses the latest backup at `~/backups/postgresql/zili_full_latest.sql`.

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

### 4. Start Grafana manually

```bash
docker run -d \
  --name grafana \
  --network zili-network \
  -e GF_SECURITY_ALLOW_EMBEDDING=true \
  -e GF_AUTH_ANONYMOUS_ENABLED=true \
  -e GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer \
  -e GF_SECURITY_ADMIN_USER=admin \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  -v ./grafana-data:/var/lib/grafana \
  -v ./provisioning:/etc/grafana/provisioning \
  -v ./dashboards:/var/lib/grafana/dashboards \
  -p 3000:3000 \
  grafana/grafana
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

## Grafana

Once running, open Grafana at:

```
http://localhost:3000
```

Default credentials: `admin` / `admin`

### Provisioning

Datasource and dashboard providers are automatically loaded from:

- `provisioning/datasources/datasource.yaml`
- `provisioning/dashboards/dashboard.yaml`

### Save dashboards

Export dashboards from Grafana UI:
1. Open dashboard → `...` → **Share** → **Export** → **Save to file**
2. Drop the JSON file into the `dashboards/` folder
3. Restart Grafana — it will auto-load on next start

### Copy dashboard to server

```bash
scp /home/sipi/Downloads/dashboard.json intercisa@homepi.local:/home/intercisa/project/go/zili-app/dashboards/
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

### Schedule backup with cron

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

### Restore from backup

Via docker exec:
```bash
docker exec -i postgresql psql -U user -d postgres < /home/intercisa/backups/postgresql/zili_full_latest.sql
```

Via docker compose profile:
```bash
docker compose --profile restore up postgresql-restore
```

---

## Access

| Service | URL |
|---|---|
| Zili App | http://localhost:8081 |
| Grafana | http://localhost:3000 |

