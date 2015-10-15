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

type Result struct {
	duration time.Duration
	err      error
	position int
	status   string
}

type Stats struct {
	ok       int
	min      time.Duration
	max      time.Duration
	nok      int
	sum_ok   time.Duration
	hostname string
}

func main() {
	// Accept one or more host names as arguments, as well as
	// a -v flag for more verbose output.

	args := os.Args[1:]
	verbosePtr := flag.Bool("v", false, "Verbosity flag")
	//waitForReply := flag.Bool("w", false, "Await replies (and wait one second) before sending next request")
	flag.Parse()
	verbose := *verbosePtr
	//wait_for_reply := *waitForReply
	args = flag.Args()
	stats_array := make([]Stats, len(args))
	fmt.Printf("Verbose mode is %t\n", verbose)
	results_channel := make(chan Result, 1)

	go summarize(results_channel, stats_array, verbose)
	for {
		for position, elem := range args {
			go func(elem string, position int) { results_channel <- ping_once(elem, verbose, position) }(elem, position)
			//go func() { results_channel <- ping_once(elem, verbose) }()
		}

		time.Sleep(time.Second)
	}

}

func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	} else {
		return b
	}
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
	} else {
		return b
	}
}

func summarize(results_channel chan Result, s []Stats, verbose bool) {
	signal_channel := make(chan os.Signal, 1)
	signal.Notify(signal_channel, syscall.SIGINT) // send sigints to signal channel

	for {
		select {
		case result := <-results_channel:
			// fixme: show this only if `verbose`
			fmt.Printf("received result: %d | %d | %s | %s\n", result.position, result.duration, result.status, result.err)
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
				fmt.Printf("%s: successful/send = %d/%d, min/avg/max = %.2f | %.2f | %.2f \n",
					stats.hostname, stats.ok, stats.nok, min_in_millisecs,
					average_in_millisecs, max_in_millisecs)
			}
			os.Exit(0)
		}
	}
}

func ping_once(hostname string, verbose bool, position int) (result Result) {
	url := "http://" + hostname

	start := time.Now()

	if verbose {
		fmt.Printf("%s: sending HTTP request... \n", hostname)
	}

	response, err := http.Get(url)
	result.err = err
	result.status = response.Status
	end := time.Now()
	result.duration = end.Sub(start) / time.Millisecond
	result.position = position

	if verbose && err != nil {
		fmt.Printf("%s: %s\n", hostname, err)
	}

	// fixme: deal with 200 OK
	return
}
