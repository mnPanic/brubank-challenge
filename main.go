package main

import (
	"fmt"
	"invoice-generator/cmd/cli"
	"invoice-generator/pkg/user"
	"log"
	"net/http"
	"os"
)

// -----------

func main() {
	finder := user.NewFinder(http.DefaultClient)
	inv, err := cli.Run(finder, os.ReadFile, os.Args[1:])
	if err != nil {
		log.Fatal(err.Error())
	}

	// Nota de diseño: Logueo esto al stderr en vez de stdout para que se pueda
	// pipear el output, pero a la vez sea un poco más ameno (que solo loguear
	// el JSON)
	fmt.Fprint(os.Stderr, "Generated invoice successfully\n")
	fmt.Println(string(inv))
}
