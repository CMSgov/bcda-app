# Database Schema Changes

## Philosophy
* All database changes will be modeled in discrete schema migrations
    * Code to roll back a migration will be included unless highly impracticable
* Migrations will receive unique sequential integer identifiers.
    * Engineers should keep aware of identifier collisions (and in some instances required migration order), and adjust their migration ID’s before merging with master, if necessary.
* Migrations will be written with production environments in mind
    * If data changes are required (default values instead of nulls, compound fields broken apart, etc.), the migration will include commands for these transformations.
    * If the migration would break running instances, reasonable effort will be made to write multiple migrations.  In the first migration, a non-breaking transition schema will be used that can support both the old and new code versions.  After all instances have been upgraded, the final version of the schema can be introduced in a second migration.
* Migrations will be tested with unit tests.  This includes the code for rolling back a migration.
* Migrations will be run as the first step of a code deployment.  Migration failure will cancel the deployment.
* Migrations will use transactions to ensure that failure rolls back the database to a valid state.
* GORM auto-migrate will be deprecated in favor of explicit migrations

## How-to
* Create migration scripts
    * Follow the philosophy above.  For example: will this schema change break production systems?  Consider making a multi-stage migration/deployment.
    * Using the file format `######_schema_name.[up|down].sql` create separate scripts for each schema change, and for reversing the schema change.
    * Add tests for both scripts in `migrations_test.go`
* Test migration scripts
    * Run `make migrations-test-ssas`