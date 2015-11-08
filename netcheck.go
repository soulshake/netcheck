package main

import (
	"flag" // see https://gobyexample.com/command-line-flags
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// define custom structure, when you get params define an array of stats
// or rather: give a pointer to the structure to the pinger function

type result struct {
	// > You don't need to export Result, type result struct.
	duration time.Duration
	err      error
	position int
	status   string
}

type stats struct {
	// You don't need to export Stats, type stats struct.
	ok       int
	min      time.Duration
	max      time.Duration
	nok      int
	sum_ok   time.Duration
	hostname string
}

// > Move the verbose pointer to the package level.
// > var verbose = flag.Bool("v". false, "explain what verbose does")

func main() {
	// Accept one or more host names as arguments, as well as
	// a -v flag for more verbose output.

	args := os.Args[1:]
	//   > Remove this line, you are overriding it at `args = flag.Args()`
	verbosePtr := flag.Bool("v", false, "Verbosity flag")
	//  > verbose := flag.Bool...
	// > We don't name pointer variables with ptr suffix in Go.

	//waitForReply := flag.Bool("w", false, "Await replies (and wait one second) before sending next request")
	flag.Parse()
	verbose := *verbosePtr
	//wait_for_reply := *waitForReply
	args = flag.Args()
	stats_array := make([]stats, len(args))
	//   > stats := make([]Stats, ...)

	fmt.Printf("Verbose mode is %t\n", verbose)
	results_channel := make(chan result, 1)
	//   > Use camel case to name the variables.
	//  > In Go, we tend to have short variable names for less verbosity.
	//  > rchan := make(chan Result, 1) would be nice here.

	go summarize(results_channel, stats_array, verbose)
	//   > go summarize(rchan, stats)
	for {
		for position, elem := range args {
			//     > for i, args := range args {}

			go func(elem string, position int) { results_channel <- ping_once(elem, verbose, position) }(elem, position)
			//       > i, args := i, args // see dynamic scope in Go,
			//  > it is weird, but we cant fix because we cant break Go 1.0.
			//  > See https://play.golang.org/p/Qi3vbrbt5J for a simpler example.
			//  > go func() {
			//  >   rchan <- pingOnce(i, elem)
			//  > }()

			//go func() { results_channel <- ping_once(elem, verbose) }()
		}

		time.Sleep(time.Second)
	}

}

func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func min(a, b time.Duration) time.Duration {
	if a == 0.0 {
		return b
	}
	if b == 0.0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func summarize(results_channel chan result, s []stats, verbose bool) {
	// > func summarize(rchan chan Result, s []Stats)

	signal_channel := make(chan os.Signal, 1)
	//   > sigCh := make(chan os.Signal, 1)

	signal.Notify(signal_channel, syscall.SIGINT) // send sigints to signal channel

	for {
		select {
		case result := <-results_channel:
			//     > case r := <- rchan:

			// fixme: show this only if `verbose`
			fmt.Printf("received result: %d | %d | %s | %s\n", result.position, result.duration, result.status, result.err)
			/*       > if r.err == nil {
			         >    s[r.position].nok++
			         >    continue
			         > }
			         > s[r.position].ok++
			         > s[r.position].min = ...*/

			if result.err == nil {
				s[result.position].ok++
				s[result.position].min = min(s[result.position].min, result.duration)
				s[result.position].max = max(s[result.position].max, result.duration)
				s[result.position].sum_ok += result.duration
			} else {
				s[result.position].nok++
			}
		case <-signal_channel:
			for _, stats := range s {
				// Print a line summarizing results for a given hostname (min/avg/max)
				min_in_millisecs := float64(stats.min * 1000000)
				max_in_millisecs := float64(stats.max * 1000000)
				average_in_nanosecs := float64(stats.sum_ok) / float64(stats.ok)
				average_in_millisecs := average_in_nanosecs * 1000000
				fmt.Println()
				//         > Do the new line as a part of the next Printf command.

				fmt.Printf("%s: successful/send = %d/%d, min/avg/max = %.2f | %.2f | %.2f \n",
					stats.hostname, stats.ok, stats.nok, min_in_millisecs,
					average_in_millisecs, max_in_millisecs)
			}
			os.Exit(0)
		}
	}
}

func ping_once(hostname string, verbose bool, position int) (result result) {
	/* > Don't name the return values unless there is no clear documentation improvement.
	   > Naked returns are always prefered over named returns.
	   > func pingOnce(hostname string, i int) Result { */

	url := "http://" + hostname

	start := time.Now()

	if verbose {
		fmt.Printf("%s: sending HTTP request... \n", hostname)
	}

	// > move `start := time.Now()` here.

	response, err := http.Get(url)
	// > resp, err := http.Get...
	result.err = err
	result.status = response.Status
	end := time.Now()
	result.duration = end.Sub(start) / time.Millisecond
	result.position = position

	/*   > if err == nil {
	>   resp.Body.Close()
	>   return
	> }
	> if verbose {
	>    fmt.Printf("%s: %s\n", hostname, err)
	> } */

	if verbose && err != nil {
		fmt.Printf("%s: %s\n", hostname, err)
	}

	// fixme: deal with 200 OK
	return
}
