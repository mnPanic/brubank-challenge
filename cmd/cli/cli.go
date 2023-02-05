package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"invoice-generator/pkg/invoice"
	"invoice-generator/pkg/invoice/call"
	"invoice-generator/pkg/platform/timeutil"
	"invoice-generator/pkg/user"
	"io"
	"strconv"
	"time"
)

// Nota de diseño: Podría haber usado un pkg como https://github.com/spf13/cobra
// para hacer el CLI, pero para este caso es overkill porque todos los
// argumentos son obligatorios, pueden ir en orden, y no hay flags.

type arguments struct {
	userTelephoneNumber string
	billingPeriodStart  string // AAAA-MM-DD
	billingPeriodEnd    string // AAAA-MM-DD
	callsCSVFileName    string
}

// FileReader reads a file from the filesystem. Used to mock reading of csv
// files.
type FileReader func(name string) ([]byte, error)

func Run(userFinder user.Finder, fileReader FileReader, rawArgs []string) (json.RawMessage, error) {
	args, err := parseArgs(rawArgs)
	if err != nil {
		return nil, fmt.Errorf("parsing arguments: %s. Usage:\n\t./invoice-generator <telephone> <billing_start> <billing_end> <calls_csv_file>", err)
	}

	billingPeriod, err := makeBillingPeriod(args.billingPeriodStart, args.billingPeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid billing period format: %s", err)
	}

	calls, err := readCalls(fileReader, args.callsCSVFileName)
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

func makeBillingPeriod(start, end string) (timeutil.Period, error) {
	const dateFormat = "2006-01-02"
	billingPeriodStart, err := time.Parse(dateFormat, start)
	if err != nil {
		return timeutil.Period{}, errors.New("invalid start date format, expected AAAA-MM-DD")
	}

	billingPeriodEnd, err := time.Parse(dateFormat, end)
	if err != nil {
		return timeutil.Period{}, errors.New("invalid end date format, expected AAAA-MM-DD")
	}

	return timeutil.Period{Start: billingPeriodStart, End: billingPeriodEnd}, nil
}

// readCalls reads the calls csv file from the specified file. Each row should
// have the following fields:
//  - Destination phone number
//  - Source phone number
//  - Duration (in seconds)
//  - Date (ISO8601 in UTC)
//
func readCalls(fileReader FileReader, path string) ([]call.Call, error) {
	// Nota de diseño: En vez de hacer os.ReadFile para leer el contenido
	// entero, podría haber hecho os.Read y leer línea por línea. Eso sería
	// más escalable para archivos más grandes que no entren en memoria.
	// Tome esta decisión por simplicidad y dado que era más sencillo de mockear
	// (os.Read devuelve *os.File).
	content, err := fileReader(path)
	if err != nil {
		return nil, fmt.Errorf("invalid csv path: %s", err)
	}

	// Columns are: numero origen,numero destino,duracion,fecha
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = 4
	reader.Read() // Skip the header column

	var calls []call.Call

	currentRow := 2 // we skipped the first one
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // finished reading the file
		}

		if err != nil {
			// csv reader errors already have line numbers, so we don't need
			// to add currentRow
			return nil, err
		}

		call, err := recordToCall(record)
		if err != nil {
			return nil, fmt.Errorf("record on line %d: %s", currentRow, err)
		}

		calls = append(calls, call)

		currentRow++
	}

	return calls, nil
}

func recordToCall(record []string) (call.Call, error) {
	sourcePhoneNumber := record[0]
	destPhoneNumber := record[1]
	duration, err := parseDuration(record[2])
	if err != nil {
		return call.Call{}, fmt.Errorf("parsing duration: %s", err)
	}

	date, err := time.Parse(timeutil.LayoutISO8601, record[3])
	if err != nil {
		return call.Call{}, fmt.Errorf("parsing date: %s", err)
	}

	return call.New(destPhoneNumber, sourcePhoneNumber, duration, date)
}

func parseDuration(rawDuration string) (uint, error) {
	// Nota: con 32 bits para segundos nos alcanza para llamadas de 8100 años,
	// así que deberíamos estar bien :P
	duration, err := strconv.ParseUint(rawDuration, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint(duration), nil
}
