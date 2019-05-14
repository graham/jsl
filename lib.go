package jsl

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/dop251/goja"
)

type IterConfig struct {
	Pre             string
	Post            string
	Iter            string
	Filter          string
	Accumulator     string
	Dedupe          string
	LibraryFilename string
	Emitter         func(interface{})
}

type Iterator interface {
	Initialize(io.Writer)
	PreIteration()
	PostIteration()
	IterFunc(interface{})
	FilterFunc(interface{}) bool
}

type StatType uint

const (
	IterOk      StatType = 0
	IterError   StatType = 1
	FilterTrue  StatType = 2
	FilterFalse StatType = 3
	FilterError StatType = 4
)

const DEFAULT_JS_CODE = `// I recommend piping this to a package.js and editing from there.

// Pre is run once at the beginning of an interation.
function pre() {
  return {};
}

// Filter is run on every row, falsy skips the 
// dedupe, iter and accumulator steps.
function filter(i, accum) {
  return true;
}

// Dedupe should return a string key, anytime a key
// is seen twice, the iter and accumulator steps are 
// skipped.
function dedupe(i) { 
  return undefined; 
}

// Return anything other than undefined and it will be emitted.
function iter(i, accum) {
  return i;
}

// Accumulator should return the updated accumulator
// object.
function accumulator(i, accum) {
  return accum;
}

// Run once at the end of the iteration.
function post(accum) { 
  return accum;
}
`

type GojaIterator struct {
	VM             *goja.Runtime
	Accumulator    goja.Value
	Emitter        func(interface{})
	dedupeMap      map[string]bool
	hasFilter      bool
	hasAccumulator bool
	hasIterator    bool
	hasDedupe      bool
}

func NewIterator(ic *IterConfig) (*GojaIterator, error) {
	iter := GojaIterator{
		dedupeMap: make(map[string]bool, 0),
	}
	iter.VM = goja.New()
	iter.Emitter = ic.Emitter

	_, err := iter.VM.RunString(DEFAULT_JS_CODE)

	if err != nil {
		panic(err)
	}

	if len(ic.Iter) > 0 {
		iter.hasIterator = true
		iter.RunString(
			fmt.Sprintf(
				"function iter(i, accum) { return %s }",
				ic.Iter,
			),
		)
	}

	if len(ic.Filter) > 0 {
		iter.hasFilter = true
		iter.RunString(
			fmt.Sprintf(
				"function filter(i, accum) { return %s }",
				ic.Filter,
			),
		)
	}

	if len(ic.Accumulator) > 0 {
		iter.hasAccumulator = true
		iter.RunString(
			fmt.Sprintf(
				"function accumulator(i, accum) { %s; return accum }",
				ic.Accumulator,
			),
		)
	}

	if len(ic.Pre) > 0 {
		iter.RunString(
			fmt.Sprintf(
				"function pre() { return %s }",
				ic.Pre,
			),
		)
	} else {
		iter.RunString("function pre() { return {} }")
	}

	if len(ic.Post) > 0 {
		iter.RunString(
			fmt.Sprintf(
				"function post(accum) { return %s }",
				ic.Post,
			),
		)
	} else {
		if iter.hasAccumulator {
			iter.RunString(
				"function post(accum) { return accum }",
			)
		} else {
			iter.RunString(
				"function post(accum) { return null }",
			)
		}
	}

	if len(ic.Dedupe) > 0 {
		iter.hasDedupe = true
		iter.RunString(
			fmt.Sprintf(
				"function dedupe(i) { return %s }",
				ic.Dedupe,
			),
		)
	}

	if len(ic.LibraryFilename) > 0 {
		iter.hasAccumulator = true
		iter.hasIterator = true
		iter.hasFilter = true
		iter.hasDedupe = true

		data, err := ioutil.ReadFile(ic.LibraryFilename)
		if err != nil {
			panic(err)
		}

		iter.RunString(string(data))
	}

	return &iter, nil
}

func (it *GojaIterator) RunString(s string) {
	it.VM.RunString(s)
}

func (it *GojaIterator) PreIteration() error {
	value, err := it.VM.RunString("pre()")
	if err != nil {
		return err
	}

	it.Accumulator = value
	return nil
}

func (it *GojaIterator) PostIteration() error {
	it.VM.Set("accum", it.Accumulator)
	value, err := it.VM.RunString("post(accum)")

	if err != nil {
		return err
	}

	it.Accumulator = value

	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}

	it.Emitter(it.Accumulator)

	return nil
}

func (it *GojaIterator) IterFunc(i interface{}) error {
	it.VM.Set("i", i)
	it.VM.Set("accum", it.Accumulator)

	keep, err := it.FilterFunc(i)

	if err != nil {
		return err
	}

	if keep == false {
		return nil
	}

	if it.hasDedupe {
		value, err := it.VM.RunString("dedupe(i)")

		if err != nil {
			return err
		}

		if goja.IsUndefined(value) == false && goja.IsNull(value) == false {
			var key string = value.String()
			if _, found := it.dedupeMap[key]; found == true {
				// We've seen this key before, skip.
				return nil
			} else {
				it.dedupeMap[key] = true
			}
		}
	}

	if it.hasIterator {
		value, err := it.VM.RunString("iter(i, accum)")

		if err != nil {
			return err
		}

		if goja.IsUndefined(value) == false {
			it.Emitter(value)
		}
	}

	if it.hasAccumulator {
		newAccum, err := it.VM.RunString("accumulator(i, accum)")
		it.Accumulator = newAccum

		if err != nil {
			return err
		}
	}

	return nil
}

func (it *GojaIterator) FilterFunc(i interface{}) (bool, error) {
	if it.hasFilter == false {
		return true, nil
	}

	it.VM.Set("i", i)
	it.VM.Set("accum", it.Accumulator)

	value, err := it.VM.RunString("filter(i, accum)")

	if err != nil {
		return false, err
	}

	return value.ToBoolean(), nil
}

func (it *GojaIterator) HandleChannel(input chan interface{}, failOnError bool) error {
	var err error
	err = it.PreIteration()
	if err != nil && failOnError {
		return err
	}

	for i := range input {
		err = it.IterFunc(i)

		if err != nil && failOnError {
			return err
		}

	}
	err = it.PostIteration()
	if err != nil && failOnError {
		return err
	}

	return nil
}
