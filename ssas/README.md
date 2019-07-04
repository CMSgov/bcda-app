# System-to-System Authentication Service (SSAS)

The SSAS can be run as a standalone web service or embedded as a library. 

# Code Organization

The service package contains a standalone http service that encapsulates the authorization library. The service also exposes a CLI for useful tasks.

Imports always go up the directory tree from leaves; that is, go files in the top-level ssas package never import from other packages in this tree. Furthermore, the ssas package and packages in plugins must not import from packages in the service directory. 

The outline below shows physical the directory structure of the code, with package names highlighted. The service and plugins directories are used for organization, but are not packages themselves. The main package is located in the service directory. Currently, the go files listed are the intended locations of code that will be migrated from bcda auth when an in-progress refactor is completed. The go files currently in place are placeholders that served to set up the code tree and illustrate the intended relationship between the service (main.go) and the ssas. 

Not shown: _test files for every .go file are assumed to be present parallel to their test subject.

- **ssas**
    - _service_
      - **cli**
        - credentials.go
          - reset-system-secret()
          - revoke-credentials()
        - group.go
          - upsert-group()
        - system.go
          - create-system()
        - token.go
          - revoke-token()
      - **api**
        - api.go
        - middleware.go
        - router.go
      - main.go
    - _plugins_
      - **alpha**
        - alpha.go
        - backend.go
      - **okta**
        - mokta.go
        - okta.go
        - oktaclient.go
        - oktajwk.go
    - provider.go
    - logger.go
    - credentials.go
      - data model and functions related to credentials
    - groups.go
      - data model and functions related to groups
    - systems.go
      - data model and functions related to systems
    - tokentools.go
      - functions related to tokens; should perhaps be renamed tokens

# Build

Build the code with `make docker-bootstrap`. Alternatively, `docker-compose up ssas` will build and run the SSAS by itself.

      

# Test

The SSAS can be tested by running `make test` or `make unit-test`.
