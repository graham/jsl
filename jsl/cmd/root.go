package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/dop251/goja"
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
var failOnException bool
var dataIsNested bool
var dataShouldFlatten bool

var outputFilename string
var inputFilename string
var appendFilename string

var stats bool

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug mode (prints to stderr)")
	RootCmd.PersistentFlags().BoolVar(&jsonEncode, "json", true, "JSON.stringify results.")
	RootCmd.PersistentFlags().BoolVar(&asText, "text", false, "Output as text, not encoded JSON.")
	RootCmd.PersistentFlags().BoolVar(&failOnException, "fail", false, "Stop iteration on uncaught javascript exception")
	RootCmd.PersistentFlags().BoolVar(&dataIsNested, "nested", false, "input is either a [] or {} and each item should be an iter step.")
	RootCmd.PersistentFlags().BoolVar(&dataShouldFlatten, "flatten", false, "flatten all sub lists [1,[2],[3,4]] -> [1,2,3,4]")

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

	RootCmd.PersistentFlags().StringVar(&appendFilename, "append", "", "append to output file instead of creating new result set.")

}

func initConfig() {
	// This function is called after parsing and before the run.

	if debugMode == false {
		log.SetOutput(ioutil.Discard)
	}

	log.Printf("Starting run %s\n", time.Now())
}

func BuildConfigFromOptions() *jsl.IterConfig {
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
		BUFFER_LEN := 0

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
		var filename string
		file_mode := os.O_CREATE | os.O_WRONLY

		if len(outputFilename) > 0 {
			filename = outputFilename
		} else if len(appendFilename) > 0 {
			filename = appendFilename
			file_mode = os.O_APPEND | os.O_CREATE | os.O_WRONLY
		}

		if len(filename) > 0 {
			outputFileHandle, err := os.OpenFile(filename, file_mode, 0644)
			if err != nil {
				panic(err)
			}
			output_writer = outputFileHandle
			defer outputFileHandle.Close()
		} else {
			output_writer = os.Stdout
		}

		output_done := make(chan bool, 0)
		go func() {
			if jsonEncode && !asText {
				enc := json.NewEncoder(output_writer)
				for i := range output_objects {
					enc.Encode(i.(goja.Value).Export())
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
					if err != nil {
						if failOnException {
							log.Println("fail", err)
							return
						} else if debugMode {
							log.Println("debug", err)
						}
					}

				}

				err = iter.PostIteration()
				if err != nil {
					panic(err)
				}

			}()
		}

		var err error
		if dataIsNested {
			err = jsl.Nested_ReadJsonObjectsUntilEOF(parsed_objects, input_reader, failOnException)
		} else if dataShouldFlatten {
			err = jsl.Flatten_ReadJsonObjectsUntilEOF(parsed_objects, input_reader, failOnException)
		} else {
			err = jsl.ReadJsonObjectsUntilEOF(parsed_objects, input_reader, failOnException)
		}

		if err != nil {
			panic(err)
		}

		wg.Wait()
		close(output_objects)
		<-output_done
	},
}
