# Development

#### Use GoReleaser to build service containers:

`goreleaser release --rm-dist --snapshot`

#### Set the latest git tag to an environment variable:

`PFDEV_TAG=$(git describe --tags $(git rev-list --tags --max-count=1) | sed 's/v//')`

#### Create an example directory structure:

```bash
mkdir -p dev/cache/zones/
touch dev/ssh-key
```

#### Spin up the compose environment:

```bash
cd dev
docker-compose up
```

#### Create test users
```bash
./dev/setup-test-users.sh
```

### Misc:

#### Drop all tables for a clean database:

`docker-compose exec -it db psql --host localhost --username api --command 'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA IF NOT EXISTS public;'`
