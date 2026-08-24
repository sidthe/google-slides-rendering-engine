// A three-slide demo of deck-engine: title slide, stat tiles + bar chart,
// and a table + code block. Run with: go run ./example [out.pptx]
package main

import (
	"fmt"
	"os"

	"github.com/sidthe/google-slides-rendering-engine/internal/demo"
)

func main() {
	out := "demo.pptx"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	d := demo.Build()
	if err := d.Save(out); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	st, _ := os.Stat(out)
	fmt.Printf("saved %s (%d bytes, %d slides)\n", out, st.Size(), d.Len())
}
