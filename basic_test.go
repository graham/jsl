package jsl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

var INPUT_TEXT string = `{"i": 0, "double": 0}
{"i": 1, "double": 2}
{"i": 2, "double": 4}
{"i": 3, "double": 6}
{"i": 4, "double": 8}
{"i": 5, "double": 10}
{"i": 6, "double": 12}
{"i": 7, "double": 14}
{"i": 8, "double": 16}
{"i": 9, "double": 18}
`

func TestIterator(t *testing.T) {
	r := strings.NewReader(INPUT_TEXT)

	var content string
	writer := bytes.NewBufferString(content)
	enc := json.NewEncoder(writer)
	iter, err := NewIterator(&IterConfig{
		Emitter: func(i interface{}) {
			enc.Encode(i)
		},
		Iter: "i.double*2",
	})

	if err != nil {
		t.Errorf("Failed to create iterator.")
	}

	reader := bufio.NewReader(r)

	var obj interface{}

	iter.PreIteration()

	for {
		var readErr error

		line, readErr := reader.ReadString('\n')

		if readErr != nil && readErr != io.EOF {
			break
		}

		if readErr == io.EOF {
			break
		}

		obj, err = LoadLine(strings.TrimSpace(line))

		if err != nil {
			t.Errorf("Failed to parse json: %s", err)
		}

		err := iter.IterFunc(obj)
		if err != nil {
			t.Errorf("Iteration failed: %s", err)
		}
	}

	if writer.String() != "0\n4\n8\n12\n16\n20\n24\n28\n32\n36\n" {
		t.Errorf("Output not correct.")
	}
}

type Payload struct {
	Value int
}

func TestAccumulator(t *testing.T) {
	results := make([]int64, 0)

	iter, err := NewIterator(&IterConfig{
		Emitter: func(i interface{}) {
			results = append(results, i.(goja.Value).ToInteger())
		},
		Iter: "i.Value * 2",
	})

	if err != nil {
		t.Errorf("Failed to create iterator.")
	}

	inputs := make(chan interface{}, 5)

	inputs <- Payload{Value: 2}
	inputs <- Payload{Value: 4}

	close(inputs)

	err = iter.HandleChannel(inputs, true)

	if err != nil {
		panic(err)
	}

	if results[0] != 4 && results[1] != 8 {
		t.Errorf("results not as expected.")
	}
}

func BenchmarkHello(b *testing.B) {
	for i := 0; i < b.N; i++ {

	}
}
