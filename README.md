JSL: JavaScript Lambda
======================

I found myself parsing through a ton of JSON at work, on my personal projects, and with almost every API I was interacting with. While grep is a fantastic tool, it fell short for me when dealing with JSON data. While grep/sed/awk are great for basic text data, I wanted something tailored for JSON, and I wrote `jsl` to provide that.

`jsl` tries to keep things simple, but powerful (a surprisingly difficult combo to engineer).

----

## Attribution

This project wouldn't be possible without the amazing [goja](https://github.com/dop251/goja) go package. It's fast, easy to use, well written and does most of the work for `jsl`. I highly recommend you check it out.

----

| The most important thing to remember is that JSL is silent on javascript errors unless you specify debug, this is because often, the schema isn't uniform, and accessors fail. If you're not seeing the output you want, make sure you use debug.

I'll add a `--fail` so that you can hard fail on javascript errors. #todo

# Help
JSL provides decent command line options and descriptions:

```
iterate over json data and run javascript on it.

Usage:
  jsl [flags]
  jsl [command]

Available Commands:
  help        Print the help
  help        Help about any command
  version     Print the version number of jsl

Flags:
      --accum string    javascript to run on every iteration (i is iter variable)
      --append          append to output file instead of creating new result set.
      --debug           enable debug mode (prints to stderr)
      --dedupe string   extract key and only emit result for key once.
      --filter string   filter out falsy results, pass truthy rows to iter
  -h, --help            help for jsl
      --input string    input filename for results (default stdin)
      --iter string     javascript to run on every iteration (i is iter variable)
      --json            JSON.stringify results. (default true)
      --output string   output filename for results (default stdout)
      --post string     code to run on the accumulator at end of iteration.
      --pre string      code to run before the iterations starts (setup accumulator)
      --src string      preload javascript file into vm
      --text            Output as text, not encoded JSON.

Use "jsl [command] --help" for more information about a command.
```

From simple to complex
======================

For now this documentation will be ok, but not great, I'll keep working on documentation and tests; and then performance.

## Pre
`--pre` allows you to define a function that will run prior to the iteration, an object you return from `pre` will be available to most functions as "accum" (the accumulator).

## Filter
`--filter` runs first for every iteration, truthy values continue the evaluation, falsy values skip and move on to the next iterable.

## Dedupe
`--dedupe` should return a _string_ key that will be used to dedupe rows with the same key, matches will result in the current iterable being skipped.

## Iter
`--iter` assuming your iterable hasn't been filtered or deduped, iter will run, results of iter that are not undefined will be emitted (either as text or json depending on your configuration).

## Accum
`--accum` can be used to record information in the accumulator per iteration, this can be helpful when building a result from your iterables rather than doing work on each of them.

## Post
`--post` post is run when the iteration has completed (no more data to read), 

## input and output
You can configure an input or output file, not setting these will result in stdin and stout being used.

## debug
Debug will likely flood your screen, but it can be helpful if youre javascript is throwing exceptions.

## --json or --text
These flags allow you to determine how the results are encoded.

## --src
This one is tricky, but you can load a file into the javascript environment. This allows you to define functions for use later; `isEven` defined in a file and loaded to be used at the command line. Or you can define all of the functions that will be used. Check out `jsl help packages` for an example of what you might use.

```
// I recommend piping this to a package.js and editing from there.

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

```

# Todo

 - [x] --fail (allow javascript errors to stop iteration
 - [ ] --stats (allow for some numbers reporting)
 - [x] --append should act like --output
 - [x] test coverage for --filter
 - [x] test coverage for --dedupe
 - [ ] test coverage for --iter
 - [x] test coverage for --accum

# Feedback and Contributions
This is a small personal project, I've found it very useful. If you have an idea for improvements or would like to submit changes, feel free to do so, or open an issue.

Have a wonderful day;

graham
