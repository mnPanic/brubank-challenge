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
		Phone:   "+54911111111",
	}

	firstInternationalCall := invoice.Call{
		DestinationPhone: "+19911111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             "2022-09-05T20:52:44Z",
	}

	secondInternationalCall := invoice.Call{
		DestinationPhone: "+19911111113",
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
		nationalPhone      = "+54911111113"
		internationalPhone = "+19911111113"
	)

	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+54911111111",
		Friends: []user.PhoneNumber{
			nationalPhone,      // National friend
			internationalPhone, // International friend
		},
	}

	// We add national and international calls to strangers to verify not all
	// are taken as friends
	nationalCall := invoice.Call{
		DestinationPhone: "+54911111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             "2022-09-05T20:52:44Z",
	}

	internationalCall := invoice.Call{
		DestinationPhone: "+19911111112",
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

func TestFriendCallsAreFreeUpToTen(t *testing.T) {
	// Up to ten friend calls are free of charge, after that they have the same
	// fare as national calls.
	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+54911111111",
		Friends: []user.PhoneNumber{
			"+54911111113",
		},
	}

	friendCall := invoice.Call{
		DestinationPhone: "+54911111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             "2022-09-05T20:52:44Z",
	}

	const maxFreeFriendCalls = 10

	var calls []invoice.Call
	for i := 0; i < maxFreeFriendCalls+1; i++ {
		calls = append(calls, friendCall)
	}

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
		expectedCalls = append(expectedCalls, expectedCall{call: friendCall, cost: 0})
	}

	// Last one is not
	expectedCalls = append(expectedCalls, expectedCall{
		call: friendCall,
		cost: 2.5, // national call fare
	})

	assertInvoiceIsExpected(t, result, testUser, expectedCalls, expectedTotalSeconds{
		international: 0,
		// Friend call seconds are counted double: in national calls and friend
		// calls
		national: friendCall.Duration * 11,
		friends:  friendCall.Duration * 11,
	})
}
