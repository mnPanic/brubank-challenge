package invoice

import (
	"fmt"
	"invoice-generator/pkg/invoice/call"
	"invoice-generator/pkg/platform/timeutil"
	"invoice-generator/pkg/user"
)

type Invoice struct {
	User                       InvoiceUser   `json:"user"`
	Calls                      []InvoiceCall `json:"calls"`
	TotalInternationalSeconds  uint          `json:"total_international_seconds"`
	TotalNationalSeconds       uint          `json:"total_national_seconds"`
	TotalFriendsSeconds        uint          `json:"total_friends_seconds"`
	TotalInterplanetarySeconds uint          `json:"total_interplanetary_seconds"`
	InvoiceTotal               float64       `json:"total"`
}

type InvoiceUser struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Phone   string `json:"phone_number"`
}

type InvoiceCall struct {
	DestinationPhone string  `json:"phone_number"` // numero destino
	Duration         uint    `json:"duration"`     // duracion
	Timestamp        string  `json:"timestamp"`    // fecha y hora
	Amount           float64 `json:"amount"`       // costo
}

// Generate generates an invoice for a given user with calls.
// It finds the user with the specified number (returning an error if it fails)
// and calculates the cost for each call.
func Generate(
	userFinder user.Finder,
	userPhoneNumber string,
	billingPeriod timeutil.Period,
	calls []call.Call,
) (Invoice, error) {
	if err := call.ValidatePhoneNumber(userPhoneNumber); err != nil {
		return Invoice{}, fmt.Errorf("user phone number: %s", err)
	}

	usr, err := userFinder.FindByPhone(user.PhoneNumber(userPhoneNumber))
	if err != nil {
		return Invoice{}, fmt.Errorf("finding user: %s", err)
	}

	callProcessor := call.NewProcessor(usr, billingPeriod, []call.Promotion{
		call.NewPromotionFreeCallsToFriends(usr),
	})

	var invoiceCalls []InvoiceCall
	for _, aCall := range calls {
		callCost, skip := callProcessor.Process(aCall)
		if skip {
			continue
		}

		invoiceCalls = append(invoiceCalls, InvoiceCall{
			DestinationPhone: aCall.DestinationPhone,
			Duration:         aCall.Duration,
			Timestamp:        aCall.Date.Format(timeutil.LayoutISO8601),
			Amount:           callCost,
		})
	}

	totalAmount, totalSeconds := callProcessor.Summarize()

	return Invoice{
		User: InvoiceUser{
			Address: usr.Address,
			Name:    usr.Name,
			Phone:   string(usr.Phone),
		},
		Calls:                      invoiceCalls,
		TotalFriendsSeconds:        totalSeconds.TotalFriendsSeconds,
		TotalNationalSeconds:       totalSeconds.TotalNationalSeconds,
		TotalInternationalSeconds:  totalSeconds.TotalInternationalSeconds,
		TotalInterplanetarySeconds: totalSeconds.TotalInterplanetarySeconds,
		InvoiceTotal:               totalAmount,
	}, nil
}
