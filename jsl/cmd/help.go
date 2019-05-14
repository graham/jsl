package cmd

import (
	"fmt"

	"github.com/graham/jsl"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(helpCmd)
}

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Print the help",
	Long:  `All software has helps.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 && args[0] == "packages" {
			fmt.Println(jsl.DEFAULT_JS_CODE)
			return
		} else {

			fmt.Println(`JSL iterates over json data and allows you to run abitrary javascript on it.

Order of operations
  call Pre()
  for i in lines:
      if Filter(i) is false then skip
      key = Dedupe(i) skip if key seen before.
      call Iter(i) emit if not undefined
      call Accum(i)
  call Post() emit if user defined.

For more information about packages and external javascript files use:
  jsl help packages

Examples:
  1) Return only even numbers:
     jsl --filter="i%2==0"

  2) Return only even numbers but only emit once:
     jsl --filter="i%2==0" --dedupe="i"

  3) Filter out odd numbers, and multiply even numbers by 10.
     jsl --filter="i%2==0" --iter="i*10"

  4) Count the number of lines:
     jsl --pre="{count:0}" --accum="accum.count+=1" --post="accum.count"
`)
		}
	},
}
