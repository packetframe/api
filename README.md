# Packetframe API

## Testing

Create a new postgres container:

`docker run --name postgres -e POSTGRES_PASSWORD=api -e POSTGRES_USER=api -e POSTGRES_DB=api -p 5432:5432 -d postgres`

Drop all tables for a clean database:

`docker exec -it postgres psql --host localhost --username api --command 'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA IF NOT EXISTS public;'`
