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
- llamadas internacionales
- llamadas a amigos, gratis hasta la 11
- llamada fuera del período de facturación
- llamada con source que no es el mismo que el usuario
*/

func TestCanGenerateInvoiceForDomesticCallsToStrangers(t *testing.T) {
	testUser := user.User{
		Name:    "Antonio Banderas",
		Address: "Calle Falsa 123",
		Phone:   "+54911111111",
		// Friends: []PhoneNumber{
		// 	"+54911111112",
		// 	"+54911111113",
		// },
	}

	mockFinder := user.NewMockFinder(
		map[user.PhoneNumber]user.User{
			testUser.Phone: testUser,
		},
	)

	firstCall := invoice.Call{
		DestinationPhone: "+19911111112",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             "2022-09-05T20:52:44Z",
	}

	secondCall := invoice.Call{
		DestinationPhone: "+19911111113",
		SourcePhone:      string(testUser.Phone),
		Duration:         60,
		Date:             "2022-09-05T20:52:44Z",
	}

	result, err := invoice.Generate(
		mockFinder,
		string(testUser.Phone),
		"2022-01-01", "2023-12-31",
		[]invoice.Call{firstCall, secondCall},
	)
	require.NoError(t, err)

	expected := invoice.Invoice{
		User: invoice.User{
			Address: "Calle Falsa 123",
			Name:    "Antonio Banderas",
			Phone:   string(testUser.Phone),
		},
		Calls: []invoice.InvoiceCall{
			{
				DestinationPhone: firstCall.DestinationPhone,
				Duration:         firstCall.Duration,
				Timestamp:        firstCall.Date,
				Amount:           float64(firstCall.Duration),
			},
		},
		TotalInternationalSeconds: 120,
		TotalNationalSeconds:      0,
		TotalFriendsSeconds:       0,
	}

	assert.Equal(t, expected, result)
}
