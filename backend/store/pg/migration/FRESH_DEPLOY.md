# Fresh Deploy SQL

For brand-new database provisioning, use:

- `backend/store/pg/migration/full_fresh_deploy.sql`

## Steps

1. Create an empty PostgreSQL database.
2. Import the merged SQL:

```bash
psql -U <user> -d <db_name> -f backend/store/pg/migration/full_fresh_deploy.sql
```

3. Start PandaWiki services normally.

The merged SQL sets `schema_migrations` to the current version (`39`, `dirty=false`) for fresh installs.

## Important

- This file is only for brand-new deployment.
- Incremental upgrade path still relies on versioned files in `backend/store/pg/migration/*.up.sql` and `*.down.sql`.
