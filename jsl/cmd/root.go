package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/graham/jsl"
	"github.com/spf13/cobra"
)

var iterCode string
var accumCode string
var preCode string
var postCode string
var filterCode string
var srcFilename string
var dedupeCode string
var wrapCode string

var debugMode bool
var jsonEncode bool
var asText bool
var parallelExecution bool

var outputFilename string
var inputFilename string
var appendToOutput bool

var stats bool

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug mode (prints to stderr)")
	RootCmd.PersistentFlags().BoolVar(&jsonEncode, "json", true, "JSON.stringify results.")
	RootCmd.PersistentFlags().BoolVar(&asText, "text", false, "Output as text, not encoded JSON.")
	//RootCmd.PersistentFlags().BoolVar(&parallelExecution, "par", false, "Parallel execution (does not preserve order).")

	// command line options for code.
	RootCmd.PersistentFlags().StringVar(&iterCode, "iter", "", "javascript to run on every iteration (i is iter variable)")
	RootCmd.PersistentFlags().StringVar(&accumCode, "accum", "", "javascript to run on every iteration (i is iter variable)")
	RootCmd.PersistentFlags().StringVar(&preCode, "pre", "", "code to run before the iterations starts (setup accumulator)")
	RootCmd.PersistentFlags().StringVar(&postCode, "post", "", "code to run on the accumulator at end of iteration.")
	RootCmd.PersistentFlags().StringVar(&filterCode, "filter", "", "filter out falsy results, pass truthy rows to iter")
	RootCmd.PersistentFlags().StringVar(&dedupeCode, "dedupe", "", "extract key and only emit result for key once.")
	RootCmd.PersistentFlags().StringVar(&srcFilename, "src", "", "preload javascript file into vm")

	RootCmd.PersistentFlags().StringVar(&outputFilename, "output", "", "output filename for results (default stdout)")
	RootCmd.PersistentFlags().StringVar(&inputFilename, "input", "", "input filename for results (default stdin)")

	RootCmd.PersistentFlags().BoolVar(&appendToOutput, "append", false, "append to output file instead of creating new result set.")
}

func initConfig() {
	// This function is called after parsing and before the run.

	if debugMode == false {
		log.SetOutput(ioutil.Discard)
	}

	log.Printf("Starting run %s\n", time.Now())
}

func BuildConfigFromOptions() *jsl.IterConfig {
	if accumCode == "" && iterCode == "" {
		iterCode = "i"
	}

	return &jsl.IterConfig{
		Iter:            iterCode,
		Accumulator:     accumCode,
		Filter:          filterCode,
		Pre:             preCode,
		Post:            postCode,
		Dedupe:          dedupeCode,
		LibraryFilename: srcFilename,
	}
}

var RootCmd = &cobra.Command{
	Use:   "jsl",
	Short: "iterate over json data and run javascript on it.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		BUFFER_LEN := 50

		config := BuildConfigFromOptions()

		// Start the reader.

		var input_reader io.Reader
		if len(inputFilename) > 0 {
			if _, err := os.Stat(inputFilename); os.IsNotExist(err) {
				fmt.Errorf("input file %s does not exist.", inputFilename)
			}

			fh, err := os.Open(inputFilename)
			if err != nil {
				panic(err)
			}
			defer fh.Close()
			input_reader = fh
		} else {
			log.Printf("Reading from stdin...")
			input_reader = os.Stdin
		}

		parsed_objects := make(chan interface{}, BUFFER_LEN)
		// Reader is ready.

		// iterators read and pass valid objects to output
		output_objects := make(chan interface{}, BUFFER_LEN)

		config.Emitter = func(i interface{}) {
			output_objects <- i
		}

		var output_writer io.Writer

		if len(outputFilename) > 0 {
			mode := os.O_CREATE | os.O_WRONLY

			if appendToOutput {
				mode = os.O_APPEND | os.O_CREATE | os.O_WRONLY
			}

			outputFileHandle, err := os.OpenFile(outputFilename, mode, 0644)
			if err != nil {
				panic(err)
			}
			defer outputFileHandle.Close()
			output_writer = bufio.NewWriter(outputFileHandle)
		} else {
			output_writer = os.Stdout
		}

		output_done := make(chan bool, 0)
		go func() {
			if jsonEncode && !asText {
				enc := json.NewEncoder(output_writer)
				for i := range output_objects {
					enc.Encode(i)
				}
			} else {
				for i := range output_objects {
					fmt.Fprintln(output_writer, i)
				}
			}
			close(output_done)
		}()
		// done with handling output of iterator and sending to stdout.

		// Lets do the actual processing.
		WORKER_COUNT := 1
		if parallelExecution {
			WORKER_COUNT = runtime.NumCPU()
		}

		wg := sync.WaitGroup{}

		for i := 0; i < WORKER_COUNT; i += 1 {
			wg.Add(1)
			go func() {
				defer wg.Done()

				iter, err := jsl.NewIterator(config)

				if err != nil {
					panic(err)
				}

				err = iter.PreIteration()

				if err != nil {
					panic(err)
				}

				for obj := range parsed_objects {
					err := iter.IterFunc(obj)
					if debugMode && err != nil {
						log.Println(err)
					}

				}

				err = iter.PostIteration()
				if err != nil {
					panic(err)
				}

			}()
		}

		jsl.ReadJsonObjectsUntilEOF(parsed_objects, input_reader)
		wg.Wait()
		close(output_objects)
		<-output_done
	},
}
