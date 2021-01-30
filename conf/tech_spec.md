# Technical Specification

## Topic: Optimizing environment variables (EVs) ACO-IP

### Scenario

    The application calls the OS for EVs throughout our codebase. From "main.go" to the source files that are entrenched deep in some package. Some of these OS calls are made repeatly when an AOC hits the API. There are a couple of concerns with this approach:

    1. OS calls for EVs are outside the process of the api, and repeated calls may be expensive.
    2. Since EVs are mutable and called continuously during the runtime of the api, a change in EVs anytime during runtime can the state of the api.

### Solution

    Move all the EVs into a single configuration file, and have the api pull in this file into memory during the start. If possible:

    1. The api should make one io call to read the file.
    2. Should be stored in memory once, and copies of it should be avoided.
    3. Dev should only able to change configuration stored in memory in go test and nowhere else.

### Plan

    1. Solution will be implemented in an iteration: bcda-app, bcda-sass-app, and etc.
    2. Solution will be rolled out for test / dev first; and then after thorough testing and coordination with SRE team, roll out to prod.
    4. The name of the solution package will be "config," and it will live in the "bcda" directory.
    5. One external package called Viper used to read in configuration file with extension "env."
    6. Viper package is encapulated by the config package, and making one two functions public: Getenv an Setenv.
    7. Setenv is only to be used in test, and it explicitly requires the test struct as a safeguard.

### Technical Details

    1. Golang reads and compiles a package once into the binary, regarless how many times the package is imported throughout the project.
    2. This means, declaring a global struct in the "config" package will allocate the data into memory once and the struct is available throughout the project.
    3. If the initiation of the (Viper) struct is done in an func init(), the configuration file should be read into memory once at the start of the application.
    4. There will be no need for any additional help from the standard library, such as sync and etc.
    5. A few locations will be searched for the configuration file. If the file doesn't exist or the file cannot be loaded, the package will default to EV.
