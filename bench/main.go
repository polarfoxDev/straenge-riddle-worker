// Command bench runs the riddle generator on its own, without Redis or the worker loop,
// so its running time can be compared against other implementations on the same instances.
//
// It mirrors generateRiddleSingleTry and the retry loop of generateRiddle, but stays on one
// goroutine: the point is to measure the algorithm, not the machine's core count.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"straenge-riddle-worker/m/convert"
	"straenge-riddle-worker/m/models"
	"straenge-riddle-worker/m/random"

	"github.com/sirupsen/logrus"
)

func main() {
	poolPath := flag.String("pool", "", "word list, one word per line, '#super X' marks a super solution")
	super := flag.String("super", "", "super solution to place")
	seed := flag.Int64("seed", 0, "random seed")
	budget := flag.Duration("budget", 10*time.Second, "wall clock budget")
	simplePaths := flag.Bool("simple-paths", false, "quantify the ambiguity check over simple paths (bug fixed)")
	emit := flag.Bool("emit", false, "print the generated riddle as JSON on stdout")
	flag.Parse()

	logrus.SetLevel(logrus.PanicLevel)
	models.SimplePathAmbiguityCheck = *simplePaths
	random.Seed(*seed)

	words, err := readPool(*poolPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	deadline := time.Now().Add(*budget)
	start := time.Now()
	attempts := 0

	for {
		if time.Now().After(deadline) {
			report(false, attempts, time.Since(start))
			return
		}
		attempts++

		riddle, err := models.NewRiddle(*super, words)
		if err != nil {
			continue
		}
		riddle, err = riddle.FillWithWords()
		if err != nil {
			continue
		}
		if ambiguous, _ := riddle.CheckForAmbiguity(); ambiguous {
			continue
		}

		elapsed := time.Since(start)
		if *emit {
			out, _ := json.Marshal(convert.TransformToOutputFormat(riddle, "benchmark"))
			fmt.Println(string(out))
		}
		report(true, attempts, elapsed)
		return
	}
}

func report(solved bool, attempts int, elapsed time.Duration) {
	fmt.Fprintf(os.Stderr, "solved=%t attempts=%d ms=%.3f\n",
		solved, attempts, float64(elapsed.Nanoseconds())/1e6)
}

func readPool(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var words []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#super ") {
			continue
		}
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line != "" {
			words = append(words, line)
		}
	}
	return words, nil
}
