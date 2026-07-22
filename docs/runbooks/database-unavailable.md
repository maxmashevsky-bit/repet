# Runbook: Database Unavailable

1. Check `/health/live`; if live is up and `/health/ready` fails, the process is running but dependencies are unavailable.
2. Inspect PostgreSQL container health and recent logs.
3. Confirm `DATABASE_URL` points to the local development database and no production credentials are used.
4. Restore service after database recovery; do not drop volumes unless explicitly resetting local data.

