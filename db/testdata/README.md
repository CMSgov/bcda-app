# Database Seed Data

## Purpose
* This folder is intended for seeding a database test container *only* with the minimum required data to run in a clean state.
* This folder should not contain any sql or data files specific to any test suite.


## Where To Put Test Specific Data
* Test specific data should go into a folder within the package called "testdata" and can be used in the database by creating a test data container and calling the function ExecuteFile() or ExecuteDir().
* Test data files should have the same name as the test. If a test is table driven and requires different seeds for each test, please create a subdirectory under "/testdata/" and give it the name of the test.
* If the test data required can be executed in a single script, a sub directory is not necessary and a file with the same name as the test can be created in the "testdata/" directory.

```
func(t *testing.T) TestFooBar() {
    assertEqual("foo", "bar")
}

func(t *testing.T) TestJobStatuses() {
    tests := []struct {
            name string      
            result bool
        }{
            {"Job Status Expired", false},
            {"Job Status Completed", false},
        }
}
```

```
|-- testdata/
|   |-- TestFooBar.sql
|   |-- TestJobStatuses/
|       |-- JobStatusExpired.sql
|       |-- JobStatusCompleted.sql
```
