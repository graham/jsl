package jsl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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

type InputObject struct {
	I      int
	Double int
	Name   string
}

func GojaValueToStruct(value *goja.Object, i interface{}) error {
	var b []byte
	var err error

	b, err = json.Marshal(value)

	if err != nil {
		return err
	}

	err = json.Unmarshal(b, i)

	if err != nil {
		return err
	}

	return nil
}

func TestIteratorWithString(t *testing.T) {
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

func TestIterator_WithFilter(t *testing.T) {
	var results []interface{} = []interface{}{}

	iter, _ := NewIterator(&IterConfig{
		Emitter: func(i interface{}) {
			results = append(results, i)
		},
		Filter: "i.Double < 10",
		Iter:   "i",
	})

	iter.PreIteration()

	for i := 0; i < 10; i += 1 {
		err := iter.IterFunc(InputObject{I: i, Double: i * 2})
		if err != nil {
			t.Errorf("iteration failed.")
		}
	}

	iter.PostIteration()

	if len(results) != 5 {
		fmt.Println(results)
		t.Errorf("Incorrect result length.")
	}
}

func TestIterator_WithDedupe(t *testing.T) {
	var results []interface{} = []interface{}{}

	iter, _ := NewIterator(&IterConfig{
		Emitter: func(i interface{}) {
			results = append(results, i)
		},
		Dedupe: "i.I % 2",
		Iter:   "i",
	})

	iter.PreIteration()

	for i := 0; i < 10; i += 1 {
		err := iter.IterFunc(InputObject{I: i, Double: i * 2})
		if err != nil {
			t.Errorf("iteration failed.")
		}
	}

	iter.PostIteration()

	if len(results) != 2 {
		t.Errorf("Incorrect result length.")
	}

	var target0 InputObject
	if err := GojaValueToStruct(results[0].(*goja.Object), &target0); err != nil {
		fmt.Println(err)
		t.Errorf("Goja value coerce failure.")
	}

	var target1 InputObject
	if err := GojaValueToStruct(results[1].(*goja.Object), &target1); err != nil {
		fmt.Println(err)
		t.Errorf("Goja value coerce failure.")
	}
}

func TestIterator_WithAccum(t *testing.T) {
	var results []interface{} = []interface{}{}

	iter, _ := NewIterator(&IterConfig{
		Emitter: func(i interface{}) {
			results = append(results, i)
		},
		Pre:         "{count:0}",
		Post:        "accum.count",
		Accumulator: "accum.count+=i.Double",
	})

	iter.PreIteration()

	for i := 0; i < 10; i += 1 {
		err := iter.IterFunc(InputObject{I: i, Double: i * 2})
		if err != nil {
			t.Errorf("iteration failed.")
		}
	}

	iter.PostIteration()

	if len(results) != 1 {
		t.Errorf("Incorrect result length.")
	}

	if results[0].(goja.Value).ToInteger() != 90 {
		t.Errorf("Sum Accumulator failed.")
	}
}

// func BenchmarkHello(b *testing.B) {
// 	for i := 0; i < b.N; i++ {

// 	}
// }
