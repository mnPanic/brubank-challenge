package invoice_test

import (
	"invoice-generator/pkg/invoice"
	"invoice-generator/pkg/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
TODO:
- llamada fuera del período de facturación
- llamada con source que no es el mismo que el usuario
*/

func TestCanGenerateInvoiceForInternationalCallsToStrangers(t *testing.T) {
	// When generating an invoice that contains international calls to
	// strangers, they should cost $1 per second.

	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
	}

	firstInternationalCall := invoice.Call{
		DestinationPhone: "+1991111111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             "2022-09-05T20:52:44Z",
	}

	secondInternationalCall := invoice.Call{
		DestinationPhone: "+1991111111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             "2022-09-05T20:52:44Z",
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		"2022-01-01", "2023-12-31",
		[]invoice.Call{firstInternationalCall, secondInternationalCall},
	)
	require.NoError(t, err)

	assertInvoiceIsExpected(t, result, testUser,
		[]expectedCall{
			{call: firstInternationalCall, cost: float64(firstInternationalCall.Duration)},
			{call: secondInternationalCall, cost: float64(secondInternationalCall.Duration)},
		},
		expectedTotalSeconds{
			international: firstInternationalCall.Duration + secondInternationalCall.Duration,
			national:      0,
			friends:       0,
		},
	)
}

func TestCallsToFriends(t *testing.T) {
	// When generating an invoice that has a call to a friend, it should be also
	// counted as a national or international call.
	const (
		nationalPhone      = "+5491111111113"
		internationalPhone = "+1991111111113"
	)

	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
		Friends: []user.PhoneNumber{
			nationalPhone,      // National friend
			internationalPhone, // International friend
		},
	}

	// We add national and international calls to strangers to verify not all
	// are taken as friends
	nationalCall := invoice.Call{
		DestinationPhone: "+5491111111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             "2022-09-05T20:52:44Z",
	}

	internationalCall := invoice.Call{
		DestinationPhone: "+1991111111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             "2022-09-05T20:52:44Z",
	}

	nationalFriendCall := invoice.Call{
		DestinationPhone: nationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             "2022-09-05T20:52:44Z",
	}

	internationalFriendCall := invoice.Call{
		DestinationPhone: internationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             "2022-09-05T20:52:44Z",
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		"2022-01-01", "2023-12-31",
		[]invoice.Call{
			nationalCall,
			internationalFriendCall,
			nationalFriendCall,
			internationalCall,
		},
	)
	require.NoError(t, err)

	assertInvoiceIsExpected(t, result, testUser,
		[]expectedCall{
			{call: nationalCall, cost: 2.5},
			{call: internationalFriendCall, cost: 0},
			{call: nationalFriendCall, cost: 0},
			{call: internationalCall, cost: float64(internationalCall.Duration)},
		},
		// Friend call seconds are counted double: as national/international and
		// friends
		expectedTotalSeconds{
			national:      nationalCall.Duration + nationalFriendCall.Duration,
			international: internationalFriendCall.Duration + internationalCall.Duration,
			friends:       nationalFriendCall.Duration + internationalFriendCall.Duration,
		},
	)
}

func TestFriendCallsAreFreeUpToTen(t *testing.T) {
	// Up to ten friend calls are free of charge, after that they have the same
	// fare as a normal call
	const (
		nationalPhone      = "+5491111111113"
		internationalPhone = "+1991111111113"
	)

	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
		Friends: []user.PhoneNumber{nationalPhone, internationalPhone},
	}

	nationalFriendCall := invoice.Call{
		DestinationPhone: nationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             "2022-09-05T20:52:44Z",
	}

	internationalFriendCall := invoice.Call{
		DestinationPhone: internationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             "2022-09-05T20:52:44Z",
	}

	const maxFreeFriendCalls = 10

	var calls []invoice.Call
	for i := 0; i < maxFreeFriendCalls; i++ {
		calls = append(calls, nationalFriendCall)
	}

	// Add the extra calls
	calls = append(calls, nationalFriendCall, internationalFriendCall)

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		"2022-01-01", "2023-12-31",
		calls,
	)
	require.NoError(t, err)

	var expectedCalls []expectedCall
	// First ten are free
	for i := 0; i < maxFreeFriendCalls; i++ {
		expectedCalls = append(expectedCalls, expectedCall{call: nationalFriendCall, cost: 0})
	}

	// Last ones are not
	expectedCalls = append(expectedCalls,
		expectedCall{call: nationalFriendCall, cost: 2.5},
		expectedCall{call: internationalFriendCall, cost: float64(internationalFriendCall.Duration)},
	)

	assertInvoiceIsExpected(t, result, testUser, expectedCalls, expectedTotalSeconds{
		international: internationalFriendCall.Duration,
		// Friend call seconds are counted double: in national calls and friend
		// calls
		national: nationalFriendCall.Duration * 11,
		friends:  nationalFriendCall.Duration*11 + internationalFriendCall.Duration,
	})
}

func TestInvalidCallFormatsReturnError(t *testing.T) {
	// If any calls have problems, we return an error instead of ignoring them.
	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
	}

	tests := map[string]struct {
		calls                []invoice.Call
		expectedErrorMessage string
	}{
		"destination phone with too few digits": {
			calls: []invoice.Call{
				{
					DestinationPhone: "+1991111111",
					SourcePhone:      string(testUser.Phone),
					Duration:         60,
					Date:             "2022-09-05T20:52:44Z",
				},
			},
			expectedErrorMessage: "invalid call #0: invalid destination phone format, should match \\+[0-9]{13}",
		},
		"source phone with too few digits": {
			// This test also verifies that calls are not filtered by phone
			// before error checking (which could lead to missed errors)
			calls: []invoice.Call{
				{
					DestinationPhone: "+1991111111111",
					SourcePhone:      "+199111111",
					Duration:         60,
					Date:             "2022-09-05T20:52:44Z",
				},
			},
			expectedErrorMessage: "invalid call #0: invalid source phone format, should match \\+[0-9]{13}",
		},
		// TODO: Fecha inválida y duración inválida (tal vez quedan del lado del
		// parseo del CSV)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := invoice.Generate(
				user.NewMockFinderForUser(testUser),
				string(testUser.Phone),
				"2022-01-01", "2023-12-31",
				tc.calls,
			)
			assert.EqualError(t, err, tc.expectedErrorMessage)
		})
	}
}

func TestCallsOutsideBillingPeriodAreIgnored(t *testing.T) {}
func TestCallsFromOtherUserAreIgnored(t *testing.T)        {}

type expectedCall struct {
	call invoice.Call
	cost float64
}

type expectedTotalSeconds struct {
	international int
	national      int
	friends       int
}

// assertInvoiceIsExpected asserts that the invoice has the expected user and
// the calls are in the specified order along with their costs and summed
// durations. The expected total is always the sum of all expected costs.
//
// Nota de diseño: Esta función la hice para alivianar tener que construir el
// struct de Invoice en todos los tests, que terminaba repitiendo mucho código.
func assertInvoiceIsExpected(t *testing.T, actualInvoice invoice.Invoice, expectedUser user.User, expectedCalls []expectedCall, expectedSeconds expectedTotalSeconds) {
	var expectedInvoiceCalls []invoice.InvoiceCall
	var expectedTotal float64

	for _, expectedCall := range expectedCalls {
		expectedInvoiceCalls = append(expectedInvoiceCalls, invoice.InvoiceCall{
			DestinationPhone: expectedCall.call.DestinationPhone,
			Duration:         expectedCall.call.Duration,
			Timestamp:        expectedCall.call.Date,
			Amount:           expectedCall.cost,
		})

		expectedTotal += expectedCall.cost
	}

	expectedInvoice := invoice.Invoice{
		User: invoice.InvoiceUser{
			Address: expectedUser.Address,
			Name:    expectedUser.Name,
			Phone:   string(expectedUser.Phone),
		},
		Calls:                     expectedInvoiceCalls,
		TotalInternationalSeconds: expectedSeconds.international,
		TotalNationalSeconds:      expectedSeconds.national,
		TotalFriendsSeconds:       expectedSeconds.friends,
		InvoiceTotal:              expectedTotal,
	}

	assert.Equal(t, expectedInvoice, actualInvoice)
}
