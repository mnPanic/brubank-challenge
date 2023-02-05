package invoice_test

import (
	"fmt"
	"invoice-generator/pkg/invoice"
	"invoice-generator/pkg/invoice/call"
	"invoice-generator/pkg/platform/timeutil"
	"invoice-generator/pkg/user"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data common to all tests

// Time periods
var (
	_timePeriod = timeutil.Period{
		Start: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2022, time.December, 31, 0, 0, 0, 0, time.UTC),
	}

	_dateInPeriod = "2022-09-05T20:52:44Z"
	_timeInPeriod = mustParse(time.RFC3339, _dateInPeriod)

	_dateOutsidePeriod = "2023-09-05T20:52:44Z"
	_timeOutsidePeriod = mustParse(time.RFC3339, _dateOutsidePeriod)
)

func TestCanGenerateInvoiceForInternationalCallsToStrangers(t *testing.T) {
	// When generating an invoice that contains international calls to
	// strangers, they should cost $1 per second.

	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
	}

	firstInternationalCall := call.Call{
		DestinationPhone: "+1991111111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             _timeInPeriod,
	}

	secondInternationalCall := call.Call{
		DestinationPhone: "+1991111111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		_timePeriod,
		[]call.Call{firstInternationalCall, secondInternationalCall},
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
	nationalCall := call.Call{
		DestinationPhone: "+5491111111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             _timeInPeriod,
	}

	internationalCall := call.Call{
		DestinationPhone: "+1991111111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             _timeInPeriod,
	}

	nationalFriendCall := call.Call{
		DestinationPhone: nationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	internationalFriendCall := call.Call{
		DestinationPhone: internationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		_timePeriod,
		[]call.Call{
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

	nationalFriendCall := call.Call{
		DestinationPhone: nationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	internationalFriendCall := call.Call{
		DestinationPhone: internationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             _timeInPeriod,
	}

	const maxFreeFriendCalls = 10

	var calls []call.Call
	for i := 0; i < maxFreeFriendCalls; i++ {
		calls = append(calls, nationalFriendCall)
	}

	// Add the extra calls
	calls = append(calls, nationalFriendCall, internationalFriendCall)

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		_timePeriod,
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

func TestCallsOutsideBillingPeriodAreIgnored(t *testing.T) {
	// There can be calls outside of the specified billing period, and they
	// shobe ignored.

	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
	}

	callOutsidePeriod := call.Call{
		DestinationPhone: "+1991111111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             _timeOutsidePeriod,
	}

	nationalCallInsidePeriod := call.Call{
		DestinationPhone: "+5491111111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		_timePeriod,
		[]call.Call{callOutsidePeriod, nationalCallInsidePeriod},
	)
	require.NoError(t, err)

	assertInvoiceIsExpected(t, result, testUser,
		[]expectedCall{
			// shouldn't contain the call outside of the period
			{call: nationalCallInsidePeriod, cost: 2.5},
		},
		expectedTotalSeconds{
			international: 0, // shouldn't be counted for seconds either
			national:      40,
			friends:       0,
		},
	)
}

func TestCallsFromDifferentUserAreIgnored(t *testing.T) {
	// There can be calls from another user than the one specified, and they
	// should be ignored
	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
	}

	callFromOtherUser := call.Call{
		DestinationPhone: "+1991111111112",
		SourcePhone:      "+5491111111112",
		Duration:         60,
		Date:             _timeInPeriod,
	}

	nationalCallFromUser := call.Call{
		DestinationPhone: "+5491111111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		_timePeriod,
		[]call.Call{callFromOtherUser, nationalCallFromUser},
	)
	require.NoError(t, err)

	assertInvoiceIsExpected(t, result, testUser,
		[]expectedCall{
			// shouldn't contain the call from other user
			{call: nationalCallFromUser, cost: 2.5},
		},
		expectedTotalSeconds{
			international: 0, // shouldn't be counted for seconds either
			national:      40,
			friends:       0,
		},
	)

}

func TestInvalidPhoneNumberShouldReturnAnError(t *testing.T) {
	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
	}

	_, err := invoice.Generate(user.NewMockFinderForUser(testUser), "invalido", _timePeriod, []call.Call{})
	assert.EqualError(t, err, "user phone number: invalid format, should match \\+[0-9]{12,13}")
}

func TestUserNotFoundShouldReturnAnError(t *testing.T) {
	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
	}

	// Different phone number than configured
	_, err := invoice.Generate(user.NewMockFinderForUser(testUser), "+5491111111112", _timePeriod, []call.Call{})
	assert.EqualError(t, err, "finding user: user not found")
}

type expectedCall struct {
	call call.Call
	cost float64
}

type expectedTotalSeconds struct {
	international uint
	national      uint
	friends       uint
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
			Timestamp:        expectedCall.call.Date.Format(timeutil.LayoutISO8601),
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

func mustParse(layout string, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(fmt.Sprintf("couldn't parse time: %s", err))
	}

	return t
}
