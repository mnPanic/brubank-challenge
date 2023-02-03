package invoice_test

import (
	"fmt"
	"invoice-generator/pkg/invoice"
	"invoice-generator/pkg/platform/timeutil"
	"invoice-generator/pkg/user"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Validar que el teléfono que se pasa como argumento es válido

// Test data common to all tests

// Time periods
var (
	_timePeriod = invoice.TimePeriod{
		Start: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2022, time.December, 31, 0, 0, 0, 0, time.UTC),
	}

	_dateInPeriod = "2022-09-05T20:52:44Z"
	_timeInPeriod = mustParse(time.RFC3339, _dateInPeriod)

	_dateOutsidePeriod = "2023-09-05T20:52:44Z"
	_timeOutsidePeriod = mustParse(time.RFC3339, _dateOutsidePeriod)
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
		Date:             _timeInPeriod,
	}

	secondInternationalCall := invoice.Call{
		DestinationPhone: "+1991111111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		_timePeriod,
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
		Date:             _timeInPeriod,
	}

	internationalCall := invoice.Call{
		DestinationPhone: "+1991111111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             _timeInPeriod,
	}

	nationalFriendCall := invoice.Call{
		DestinationPhone: nationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	internationalFriendCall := invoice.Call{
		DestinationPhone: internationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		_timePeriod,
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
		Date:             _timeInPeriod,
	}

	internationalFriendCall := invoice.Call{
		DestinationPhone: internationalPhone,
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             _timeInPeriod,
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
					Date:             _timeInPeriod,
				},
			},
			expectedErrorMessage: "invalid call #0: invalid destination phone format, should match \\+[0-9]{12,13}",
		},
		"source phone with too few digits": {
			// This test also verifies that calls are not filtered by phone
			// before error checking (which could lead to missed errors)
			calls: []invoice.Call{
				{
					DestinationPhone: "+1991111111111",
					SourcePhone:      "+199111111",
					Duration:         60,
					Date:             _timeInPeriod,
				},
			},
			expectedErrorMessage: "invalid call #0: invalid source phone format, should match \\+[0-9]{12,13}",
		},
		// TODO: Fecha inválida y duración inválida (tal vez quedan del lado del
		// parseo del CSV)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := invoice.Generate(
				user.NewMockFinderForUser(testUser),
				string(testUser.Phone),
				_timePeriod,
				tc.calls,
			)
			assert.EqualError(t, err, tc.expectedErrorMessage)
		})
	}
}

func TestCallsOutsideBillingPeriodAreIgnored(t *testing.T) {
	// There can be calls outside of the specified billing period, and they
	// shobe ignored.

	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+5491111111111",
	}

	callOutsidePeriod := invoice.Call{
		DestinationPhone: "+1991111111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             _timeOutsidePeriod,
	}

	nationalCallInsidePeriod := invoice.Call{
		DestinationPhone: "+5491111111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		_timePeriod,
		[]invoice.Call{callOutsidePeriod, nationalCallInsidePeriod},
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

	callFromOtherUser := invoice.Call{
		DestinationPhone: "+1991111111112",
		SourcePhone:      "+5491111111112",
		Duration:         60,
		Date:             _timeInPeriod,
	}

	nationalCallFromUser := invoice.Call{
		DestinationPhone: "+5491111111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         40,
		Date:             _timeInPeriod,
	}

	result, err := invoice.Generate(
		user.NewMockFinderForUser(testUser),
		string(testUser.Phone),
		_timePeriod,
		[]invoice.Call{callFromOtherUser, nationalCallFromUser},
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

type expectedCall struct {
	call invoice.Call
	cost float64
}

// TODO: Nota de diseño: elegí poner date en expectedCall en lugar de que
// assertInvoiceIsExpected haga la conversión a iso8601 porque sino no estaría
// testeando correctamente cómo se formatean las fechas

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
			// TODO: considerar tomar directamente []InvoiceCall y que esto lo
			// defina el test
			Duration:  expectedCall.call.Duration,
			Timestamp: expectedCall.call.Date.Format(timeutil.LayoutISO8601),
			Amount:    expectedCall.cost,
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
