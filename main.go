package main

import (
	"fmt"
	"invoice-generator/cmd/cli"
	"invoice-generator/pkg/user"
	"log"
	"os"
)

// -----------

func main() {
	inv, err := cli.Run(user.NewMockFinderForUser(
		user.User{
			Name:    "Antonio Banderas",
			Address: "Calle Falsa 123",
			Phone:   "+5491167930920",
		},
	), os.Args[1:])
	if err != nil {
		log.Fatal(err.Error())
	}

	// Nota de diseño: Logueo esto al stderr en vez de stdout para que se pueda
	// pipear el output, pero a la vez sea un poco más ameno (que solo loguear
	// el JSON)
	fmt.Fprint(os.Stderr, "Generated invoice successfully\n")
	fmt.Printf(string(inv))
}
