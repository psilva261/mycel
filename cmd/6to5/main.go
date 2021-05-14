// Conversion of ES6+ into ES5.1 (wip)
//
// TODO: turn into a script that uses devjs
package main

import (
	"fmt"
	"github.com/jvatic/goja-babel"
	"io"
	"log"
	"os"
)

func Main() (err error) {
	babel.Init(1) // Setup 1 transformer (can be any number > 0)
	r, err := babel.Transform(os.Stdin, map[string]interface{}{
		"plugins": []string{
			"transform-arrow-functions",
			"transform-block-scoping",
			"transform-classes",
			"transform-destructuring",
			"transform-spread",
			"transform-parameters",
		},
	})
	if err != nil {
		return fmt.Errorf("transform: %v", err)
	}
	_, err = io.Copy(os.Stdout, r)

	return
}

func main() {
	if err := Main(); err != nil {
		log.Fatalf("%v",err)
	}
}
