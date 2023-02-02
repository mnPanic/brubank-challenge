package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"invoice-generator/pkg/invoice"
	"invoice-generator/pkg/user"
	"time"
)

// Nota de diseño: Podría haber usado un pkg como https://github.com/spf13/cobra
// para hacer el CLI, pero para este caso es overkill porque todos los
// argumentos son obligatorios, pueden ir en orden, y no hay flags.

func Run(userFinder user.Finder, rawArgs []string) (json.RawMessage, error) {
	args, err := parseArgs(rawArgs)
	if err != nil {
		return nil, fmt.Errorf("parsing arguments: %s. Usage:\n\t./invoice-generator <telephone> <billing_start> <billing_end> <calls_csv_file>")
	}

	billingPeriod, err := makeBillingPeriod(args.billingPeriodStart, args.billingPeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid billing period format: %s", err)
	}

	calls, err := readCalls(args.callsCSVFileName)
	if err != nil {
		return nil, fmt.Errorf("reading calls: %s", err)
	}

	invoice, err := invoice.Generate(userFinder, args.userTelephoneNumber, billingPeriod, calls)
	if err != nil {
		return nil, fmt.Errorf("generating invoice: %s", err)
	}

	invoiceJSON, err := json.Marshal(invoice)
	if err != nil {
		return nil, fmt.Errorf("invoice json marshal: %s", err)
	}

	return invoiceJSON, nil
}

type arguments struct {
	userTelephoneNumber string
	billingPeriodStart  string // AAAA-MM-DD
	billingPeriodEnd    string
	callsCSVFileName    string
}

func parseArgs(args []string) (arguments, error) {
	if len(args) != 4 {
		return arguments{}, errors.New("wrong number of arguments, expected 4")
	}

	return arguments{
		userTelephoneNumber: args[0],
		billingPeriodStart:  args[1],
		billingPeriodEnd:    args[2],
		callsCSVFileName:    args[3],
	}, nil
}

func makeBillingPeriod(start, end string) (invoice.TimePeriod, error) {
	const dateFormat = "2006-01-02"
	billingPeriodStart, err := time.Parse(dateFormat, start)
	if err != nil {
		return invoice.TimePeriod{}, errors.New("invalid billing period start date format, expected AAAA-MM-DD")
	}

	billingPeriodEnd, err := time.Parse(dateFormat, end)
	if err != nil {
		return invoice.TimePeriod{}, errors.New("invalid billing period end date format, expected AAAA-MM-DD")
	}

	return invoice.TimePeriod{Start: billingPeriodStart, End: billingPeriodEnd}, nil
}

func readCalls(path string) ([]invoice.Call, error) {
	return nil, nil
}
