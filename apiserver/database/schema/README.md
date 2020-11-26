# Database migrations

Database migrations must be compiled into the program. In order to accomplish this, a helper application
in this directory scans the directory for SQL files containing database migration scripts.
The contents of the SQL files are added to a string slice in the file `../zz-migrations-generated.go`.

## Adding database migrations

Create a new file in this directory, increasing the serial number and giving it a fitting title.

Your migration MUST be wrapped in a transaction statement. At the end, insert a row into the migrations
table to indicate to the program that this migration has been run.

Example below:

```postgresql
-- Start transaction
START TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;

-- Edit tables as needed
ALTER TABLE ...

-- Mark this database migration as completed. Replace xxx with the serial number of the migration.
-- You do not need to include the zeroes as this field is an integer.
INSERT INTO migrations (version, created)
VALUES (xxx, now());

-- Commit changes
COMMIT;
```

After adding or updating the SQL file, you must run the migration helper using `go generate` from the project folder:

```
$ go generate ./...
```

The database migration will be performed when the application is started by calling the `Migrate()` function.
