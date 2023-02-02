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

	mockFinder := user.NewMockFinder(
		map[user.PhoneNumber]user.User{
			testUser.Phone: testUser,
		},
	)

	result, err := invoice.Generate(
		mockFinder,
		string(testUser.Phone),
		"2022-01-01", "2023-12-31",
		[]invoice.Call{firstInternationalCall, secondInternationalCall},
	)
	require.NoError(t, err)

	expected := invoice.Invoice{
		User: invoice.InvoiceUser{
			Address: testUser.Address,
			Name:    testUser.Name,
			Phone:   string(testUser.Phone),
		},
		Calls: []invoice.InvoiceCall{
			{
				DestinationPhone: firstInternationalCall.DestinationPhone,
				Duration:         firstInternationalCall.Duration,
				Timestamp:        firstInternationalCall.Date,
				Amount:           float64(firstInternationalCall.Duration),
			},
			{
				DestinationPhone: secondInternationalCall.DestinationPhone,
				Duration:         secondInternationalCall.Duration,
				Timestamp:        secondInternationalCall.Date,
				Amount:           float64(secondInternationalCall.Duration),
			},
		},
		TotalInternationalSeconds: firstInternationalCall.Duration + secondInternationalCall.Duration,
		TotalNationalSeconds:      0,
		TotalFriendsSeconds:       0,
		InvoiceTotal:              float64(firstInternationalCall.Duration) + float64(secondInternationalCall.Duration),
	}

	assert.Equal(t, expected, result)
}

func TestFriendsAreNationalCalls(t *testing.T) {
	// When generating an invoice that has a call to a friend, it should be also
	// counted as a national call.
	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+54911111111",
		Friends: []user.PhoneNumber{
			"+54911111113",
		},
	}

	mockFinder := user.NewMockFinder(
		map[user.PhoneNumber]user.User{
			testUser.Phone: testUser,
		},
	)

	nationalCall := invoice.Call{
		DestinationPhone: "+54911111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             "2022-09-05T20:52:44Z",
	}

	friendCall := invoice.Call{
		DestinationPhone: "+54911111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             "2022-09-05T20:52:44Z",
	}

	result, err := invoice.Generate(
		mockFinder,
		string(testUser.Phone),
		"2022-01-01", "2023-12-31",
		[]invoice.Call{nationalCall, friendCall},
	)
	require.NoError(t, err)

	expected := invoice.Invoice{
		User: invoice.InvoiceUser{
			Address: testUser.Address,
			Name:    testUser.Name,
			Phone:   string(testUser.Phone),
		},
		Calls: []invoice.InvoiceCall{
			{
				DestinationPhone: nationalCall.DestinationPhone,
				Duration:         nationalCall.Duration,
				Timestamp:        nationalCall.Date,
				Amount:           2.5,
			},
			{
				DestinationPhone: friendCall.DestinationPhone,
				Duration:         friendCall.Duration,
				Timestamp:        friendCall.Date,
				Amount:           0,
			},
		},
		TotalInternationalSeconds: 0,
		// Friend call seconds are counted double: in national calls and friend
		// calls
		TotalNationalSeconds: nationalCall.Duration + friendCall.Duration,
		TotalFriendsSeconds:  friendCall.Duration,
		InvoiceTotal:         2.5,
	}

	assert.Equal(t, expected, result)
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

	mockFinder := user.NewMockFinder(
		map[user.PhoneNumber]user.User{
			testUser.Phone: testUser,
		},
	)

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
		mockFinder,
		string(testUser.Phone),
		"2022-01-01", "2023-12-31",
		calls,
	)
	require.NoError(t, err)

	var expectedCalls []invoice.InvoiceCall
	// First ten are free
	for i := 0; i < maxFreeFriendCalls; i++ {
		expectedCalls = append(expectedCalls, invoice.InvoiceCall{
			DestinationPhone: friendCall.DestinationPhone,
			Duration:         friendCall.Duration,
			Timestamp:        friendCall.Date,
			Amount:           0,
		})
	}
	// Last one is not
	expectedCalls = append(expectedCalls, invoice.InvoiceCall{
		DestinationPhone: friendCall.DestinationPhone,
		Duration:         friendCall.Duration,
		Timestamp:        friendCall.Date,
		Amount:           2.5, // national call fare
	})

	expected := invoice.Invoice{
		User: invoice.InvoiceUser{
			Address: testUser.Address,
			Name:    testUser.Name,
			Phone:   string(testUser.Phone),
		},
		Calls:                     expectedCalls,
		TotalInternationalSeconds: 0,
		// Friend call seconds are counted double: in national calls and friend
		// calls
		TotalNationalSeconds: friendCall.Duration * 11,
		TotalFriendsSeconds:  friendCall.Duration * 11,
		InvoiceTotal:         2.5,
	}

	assert.Equal(t, expected, result)
}
