## Setup
1. Follow installing go + vscode [setup guide](https://marketplace.visualstudio.com/items?itemName=golang.go#getting-started).
2. Ensure that bcda-app is found within the $GOPATH.
```
> echo $GOPATH
/Users/bcda-developer/go
```

*NOTE:* You may have to manually set the $GOPATH

```
> mkdir -p $GOPATH/src/github.com/CMSgov
> git clone git@github.com:CMSgov/bcda-app.git $GOPATH/src/github.com/CMSgov/bcda-app
```

This allows bcda-app to be built/tested locally.

## Testing
1. Run `make unit-test-db`. It ensures that the postgres db used for unit tests is seeded correctly.
2. Run package/unit level tests using vscode's inline tools