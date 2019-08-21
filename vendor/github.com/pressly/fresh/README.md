# Purpose of this fork [![License](http://img.shields.io/:license-mit-blue.svg?style=flat-square)](http://badges.mit-license.org) [![Join the chat at https://gitter.im/pilu/fresh](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/pilu/fresh?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

Development by the original Fresh creator seems to have slowed down a lot with important pull requests waiting for many months to be reviewed and merged, while we're waiting for Fresh2 to be released.

I will be cherry-picking commits from all the forks just to have a better, more up to date version. ~~Unless I stumble upon something affecting me personally I don't intend to put significant amount of time into improving this already great tool.~~ On day one I rewrote config handling, added multiple directory watching and excluding directories from being watched

I promise to be very responsive reviewing and accepting (or rejecting) pull requests.

## Rerun, an alternative to Fresh

Check out https://github.com/VojtechVitek/rerun

# Fresh

Fresh is a command line tool that builds and (re)starts your web application every time you save a Go or template file.

If the web framework you are using supports the Fresh runner, it will show build errors on your browser.

It has been tested with:
* [gocraft/web](https://github.com/gocraft/web)
* [Martini](https://github.com/codegangsta/martini)
* [Traffic](https://github.com/pilu/traffic)

## Installation

    go get -u github.com/pressly/fresh

## Usage

    cd /path/to/myapp

Start fresh:

    fresh

Fresh will watch for file events, and every time you create/modifiy/delete a file it will build and restart the application.
If `go build` returns an error, it will log it in the tmp folder.

[Traffic](https://github.com/pilu/traffic) already has a middleware that shows the content of that file if it is present. This middleware is automatically added if you run a Traffic web app in dev mode with Fresh.
Check the `_examples` folder if you want to use it with Martini or Gocraft Web.

`fresh` uses [toml](https://github.com/BurntSushi/toml) configuration files.
`./runner.conf` is loaded by default (if it exists), but you may specify an alternative config filepath using `-c`:

    fresh -c other_runner.conf

Here is a sample config file with the default settings:

    root              = "."
    tmp_path          = "./tmp"
    build_name        = "runner-build"
    build_log         = "runner-build-errors.log"
    valid_ext         = [".go", ".tpl", ".tmpl", ".html"]
    build_delay       = 600
    colors            = 1
    log_color_main    = "cyan"
    log_color_build   = "yellow"
    log_color_runner  = "green"
    log_color_watcher = "magenta"
    log_color_app     = ""

`fresh` accepts custom build flags that are passed to build command of the watched code. To add them use `-b`:

    fresh -b "--race -tags 'tag1'"

`fresh` accepts custom run arguments that are passed to built binary when starting it. To add them use `-r`:

    fresh -r "-configFile ../config/testing.conf"


## Original Author

* [Andrea Franz](http://gravityblast.com)

## Maintainter of this fork

* [Maciej Lisiewski](https://twitter.com/lisiewski)


## More

* [Mailing List](https://groups.google.com/d/forum/golang-fresh)

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request
