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
// or rather: give a pointer to the structure to the pinger function (?)

type result struct {
	duration time.Duration
	err      error
	position int
	status   string
}

type stats struct {
	ok       int
	min      time.Duration
	max      time.Duration
	nok      int
	sum_ok   time.Duration
	hostname string
}

var verbose = flag.Bool("v", false, "Display additional info")

//var waitForReply := flag.Bool("w", false, "Await replies (and wait one second) before sending next request")

func main() {
	// Accept one or more host names as arguments, as well as
	// a -v flag for more verbose output and a -w flag to wait up to 1 second
	// for a reply before proceeding.

	flag.Parse()
	verbose := *verbose
	//waitForReply := *waitForReply
	args := flag.Args()
	stats := make([]stats, len(args))

	fmt.Printf("Verbose mode is %t\n", verbose)
	rchan := make(chan result, 1)

	go summarize(rchan, stats)
	for {
		for i, elem := range args {
			go func() {
				rchan <- pingOnce(elem, i)
			}()
			//  > i, args := i, args // see dynamic scope in Go,
			//  > it is weird, but we cant fix because we cant break Go 1.0.
			//  > See https://play.golang.org/p/Qi3vbrbt5J for a simpler example. (?)
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

func summarize(rchan chan result, s []stats) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT) // send sigints to signal channel

	for {
		select {
		case result := <-rchan:

			fmt.Printf("received result: %d | %d | %s | %s\n", result.position, result.duration, result.status, result.err)
			if result.err != nil {
				s[result.position].nok++
				continue
			}
			s[result.position].ok++
			s[result.position].min = min(s[result.position].min, result.duration)
			s[result.position].max = max(s[result.position].max, result.duration)
			s[result.position].sum_ok += result.duration

		case <-sigCh:
			for _, stats := range s {
				// Print a line summarizing results for a given hostname (min/avg/max)
				min_in_millisecs := float64(stats.min * 1000000)
				max_in_millisecs := float64(stats.max * 1000000)
				average_in_nanosecs := float64(stats.sum_ok) / float64(stats.ok)
				average_in_millisecs := average_in_nanosecs * 1000000

				fmt.Printf("\n%s: successful/send = %d/%d, min/avg/max = %.2f | %.2f | %.2f \n",
					stats.hostname, stats.ok, stats.nok, min_in_millisecs,
					average_in_millisecs, max_in_millisecs)
			}
			os.Exit(0)
		}
	}
}

func pingOnce(hostname string, i int) result {
	var r result
	url := "http://" + hostname

	if *verbose {
		fmt.Printf("%s: sending HTTP request... \n", hostname)
	}

	start := time.Now()
	resp, err := http.Get(url)
	r.err = err
	r.status = resp.Status
	end := time.Now()
	r.duration = end.Sub(start) / time.Millisecond
	r.position = i

	if err == nil {
		resp.Body.Close()
		return r
	}

	fmt.Printf("%s: %s\n", hostname, err)

	// fixme: deal with 200 OK
	return r
}
