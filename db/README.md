# Test Database Container

## Purpose
To have an idempotent database for each test run.

## Implementation Strategy

`TestDatabaseContainer` is a lightweight wrapper of https://golang.testcontainers.org/modules/postgres/. Because BCDA uses pgx, pgxpool, and sql/db for database connections, this wrapper type will implement methods to utilize any of the aforementioned connection types.

This type also implements methods to apply migrations and seed the database with an initial set of necessary data for the BCDA application to execute database operations.


## How To Use

1. Create the container in the setup of the test suite; this is the longest running step.
2. Create the database connection in the setup of the test or the setup of the subtest.
3. (optional) Seed any additional test data with TestDatabaseContainer.ExecuteFile() or TestDatabaseContainer.ExecuteDir(). For details on seed data, please consult the README.md in ./seeddata
4. Restore a snapshot in the test teardown.

*Note*: Database snapshots cannot be created or restored if a database connection still exists.

```
type FooTestSuite struct {
	suite.Suite
	db          *sql.DB    // example; pgx or pool can also be used
	dbContainer db.TestDatabaseContainer.  // example; this is optional to be part of the test suite
}

func (s *FooTestSuite) SetupSuite() {
	ctr, err := db.NewTestDatabaseContainer()
	require.NoError(s.T(), err)
	s.dbContainer = ctr
}


func (s *FooTestSuite) SetupTest() {
    db, err := s.dbContainer.NewSqlDbConnection()
	require.NoError(s.T(), err)
	s.db = db
}

func (s *FooTestSuite) TearDownTest() {
	s.db.Close()
	err := s.dbContainer.RestoreSnapshot().   // example, you can restore from another desired snapshot
	require.NoError(s.T(), err)
}
```
