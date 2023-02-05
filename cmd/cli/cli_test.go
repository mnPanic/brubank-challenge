package cli_test

import (
	"errors"
	"fmt"
	"invoice-generator/cmd/cli"
	"invoice-generator/pkg/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	phone    = "+5491167950940"
	filename = "filename-doesnt-matter"
)

func TestOnInvalidArgumentsShouldReturnError(t *testing.T) {
	_, err := cli.Run(defaultUserFinder(), defaultReader(), []string{"just one arg"})
	assert.EqualError(t, err, "parsing arguments: wrong number of arguments, expected 4. Usage:\n\t./invoice-generator <telephone> <billing_start> <billing_end> <calls_csv_file>")
}

func TestShouldFailWithInvalidBillingPeriodStart(t *testing.T) {
	// Start period missing day
	_, err := cli.Run(defaultUserFinder(), defaultReader(), []string{phone, "2022-10", "2022-10-01", filename})
	assert.EqualError(t, err, "invalid billing period format: invalid start date format, expected AAAA-MM-DD")
}

func TestShouldFailWithInvalidBillingPeriodEnd(t *testing.T) {
	// End period missing day
	_, err := cli.Run(defaultUserFinder(), defaultReader(), []string{phone, "2022-10-01", "2022-10", filename})
	assert.EqualError(t, err, "invalid billing period format: invalid end date format, expected AAAA-MM-DD")
}

func TestShouldFailOnInvalidCSVPath(t *testing.T) {
	failingReader := func(_ string) ([]byte, error) {
		return nil, errors.New("not found")
	}

	_, err := cli.Run(defaultUserFinder(), failingReader, []string{phone, "2022-10-01", "2022-10-01", filename})
	assert.EqualError(t, err, "reading calls: invalid csv path: not found")
}

func TestShouldFailOnLineWithWrongNumberOfFields(t *testing.T) {
	// Line 3 doesn't have the duration field
	reader := readerWithContent(`numero origen,numero destino,duracion,fecha
	+5491167980950,+191167980952,462,2020-11-10T04:02:45Z
	+5491167980950,+191167980952,2020-11-10T04:02:45Z
	+5491167910920,+191167980952,392,2020-08-09T04:45:25Z`)

	_, err := cli.Run(defaultUserFinder(), reader, []string{phone, "2022-10-01", "2022-10-01", filename})
	assert.EqualError(t, err, "reading calls: record on line 3: wrong number of fields")
}

func TestShouldFailOnLineWithInvalidDuration(t *testing.T) {
	// In line 3, the duration field is a string instead of an int
	reader := readerWithContent(`numero origen,numero destino,duracion,fecha
	+5491167980950,+191167980952,462,2020-11-10T04:02:45Z
	+5491167980950,+191167980952,esto-no-es-duracion,2020-11-10T04:02:45Z
	+5491167910920,+191167980952,392,2020-08-09T04:45:25Z`)

	_, err := cli.Run(defaultUserFinder(), reader, []string{phone, "2022-10-01", "2022-10-01", filename})
	assert.EqualError(t, err, "reading calls: record on line 3: parsing duration: strconv.ParseUint: parsing \"esto-no-es-duracion\": invalid syntax")
}

func TestShouldFailOnLineWithInvalidDate(t *testing.T) {
	reader := readerWithContent(`numero origen,numero destino,duracion,fecha
	+5491167980950,+191167980952,462,2020-11-10T04:02:45Z
	+5491167980950,+191167980952,400,2020-11-10T:02:45Z
	+5491167910920,+191167980952,392,2020-08-09T04:45:25Z`)

	_, err := cli.Run(defaultUserFinder(), reader, []string{phone, "2022-10-01", "2022-10-01", filename})
	assert.EqualError(t, err, "reading calls: record on line 3: parsing date: parsing time \"2020-11-10T:02:45Z\" as \"2006-01-02T15:04:05Z0700\": cannot parse \":02:45Z\" as \"15\"")
}

func TestShouldFailOnLineWithInvalidDestinationNumber(t *testing.T) {
	reader := readerWithContent(`numero origen,numero destino,duracion,fecha
	+5491167980950,+191167980952,462,2020-11-10T04:02:45Z
	+5491167980950,+191167980,400,2020-11-10T04:02:45Z
	+5491167910920,+191167980952,392,2020-08-09T04:45:25Z`)

	_, err := cli.Run(defaultUserFinder(), reader, []string{phone, "2022-10-01", "2022-10-01", filename})
	assert.EqualError(t, err, "reading calls: record on line 3: destination phone: invalid format, should match \\+[0-9]{12,13}")
}

func TestShouldFailOnLineWithInvalidSourceNumber(t *testing.T) {
	reader := readerWithContent(`numero origen,numero destino,duracion,fecha
	+5491167980950,+191167980952,462,2020-11-10T04:02:45Z
	+5491167980,+5491167980950,400,2020-11-10T04:02:45Z
	+5491167910920,+191167980952,392,2020-08-09T04:45:25Z`)

	_, err := cli.Run(defaultUserFinder(), reader, []string{phone, "2022-10-01", "2022-10-01", filename})
	assert.EqualError(t, err, "reading calls: record on line 3: source phone: invalid format, should match \\+[0-9]{12,13}")
}

func TestShouldReturnInvoiceGenerationErrors(t *testing.T) {
	// Invoice generation fails with an invalid user
	_, err := cli.Run(defaultUserFinder(), defaultReader(), []string{"+5491167950941", "2022-10-01", "2022-10-01", filename})
	assert.EqualError(t, err, "generating invoice: finding user: user not found")
}

func TestReturnsGeneratedInvoice(t *testing.T) {
	reader := readerWithContent(`numero origen,numero destino,duracion,fecha
+5491167950940,+191167980952,462,2020-11-10T04:02:45Z
+5491167950940,+191167980952,392,2020-08-09T04:45:25Z
+5491167950940,+541167980953,60,2020-05-10T04:45:25Z`)

	userFinder := user.NewMockFinderForUser(
		user.User{
			Name:    "Hideo Kojima",
			Address: "Calle Falsa 123",
			Phone:   phone,
			Friends: []user.PhoneNumber{"+541167980953"},
		},
	)

	result, err := cli.Run(userFinder, reader, []string{phone, "2020-01-01", "2022-09-01", filename})
	require.NoError(t, err)

	expectedInvoice := `{
		"user": {
			"address": "Calle Falsa 123",
			"name": "Hideo Kojima",
			"phone_number": "+5491167950940"
		},
		"calls": [
			{
				"phone_number": "+191167980952",
				"duration": 462,
				"timestamp": "2020-11-10T04:02:45Z",
				"amount": 462.0
			},
			{
				"phone_number": "+191167980952",
				"duration": 392,
				"timestamp": "2020-08-09T04:45:25Z",
				"amount": 392.0
			},
			{
				"phone_number": "+541167980953",
				"duration": 60,
				"timestamp": "2020-05-10T04:45:25Z",
				"amount": 0.0
			}
		],
		"total_international_seconds":854,
		"total_national_seconds":60,
		"total_friends_seconds":60,
		"total":854
	}`
	fmt.Printf("expected: %s\nactual:%s", expectedInvoice, string(result))
	assert.JSONEq(t, expectedInvoice, string(result))
}

func defaultUserFinder() user.Finder {
	return user.NewMockFinderForUser(
		user.User{
			Name:    "Antonio Banderas",
			Address: "Calle Falsa 123",
			Phone:   phone,
		},
	)
}

func defaultReader() cli.FileReader {
	return readerWithContent(`numero origen,numero destino,duracion,fecha
+5491167980950,+191167980952,462,2020-11-10T04:02:45Z
+5491167910920,+191167980952,392,2020-08-09T04:45:25Z`)
}

func readerWithContent(content string) cli.FileReader {
	return func(name string) ([]byte, error) {
		return []byte(content), nil
	}
}
