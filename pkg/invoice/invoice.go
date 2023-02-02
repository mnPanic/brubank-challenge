package invoice

import (
	"fmt"
	"invoice-generator/pkg/user"
)

type Invoice struct {
	User                      InvoiceUser `json:"user"`
	Calls                     []InvoiceCall
	TotalInternationalSeconds int `json:"total_international_seconds"`
	TotalNationalSeconds      int
	TotalFriendsSeconds       int
	InvoiceTotal              float64
}

type InvoiceUser struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Phone   string `json:"phone_number"`
}

type InvoiceCall struct {
	DestinationPhone string  `json:"phone_number"` // numero destino
	Duration         int     `json:"duration"`     // duracion
	Timestamp        string  `json:"timestamp"`    // fecha y hora
	Amount           float64 `json:"amount"`       // costo
}

type Call struct {
	DestinationPhone string
	SourcePhone      string
	Duration         int    // seconds
	Date             string // ISO 8601 in UTC
	// TODO: cambiar a time.Duration y time.Time
}

// TODO: Tiene sentido que esto sea parte de call, o un modulo a parte
func (c Call) CalculateCost() float64 {
	// TODO: contemplar diferentes tipos de llamadas
	return float64(c.Duration)
}

type CallType int

const (
	CallTypeNational CallType = iota + 1
	CallTypeInternational
	CallTypeFriend
)

func (c Call) Type() CallType {
	return CallTypeInternational
}

// Generate generates an invoice for a given user with calls
func Generate(
	userFinder user.Finder,
	userPhoneNumber string,
	billingStartDate string,
	billingEndDate string,
	calls []Call,
) (Invoice, error) {
	usr, err := userFinder.FindByPhone(user.PhoneNumber(userPhoneNumber))
	if err != nil {
		return Invoice{}, fmt.Errorf("user not found, finder: %s", err)
	}

	// TODO: filtrar llamadas que no sean del usuario y no estén en el período
	// de facturación

	// TODO: Falopa para que quede más limpio: mapa indexado por CallType que
	// tiene como valor un int.
	var (
		totalInternationalSeconds int
		totalNationalSeconds      int
		totalFriendsSeconds       int

		totalAmount float64
	)

	// TODO: usar make para hacer más eficiente la transformación
	var invoiceCalls []InvoiceCall
	for _, call := range calls {
		// TODO: Evitar calcular dos veces el call type
		callType := call.Type()
		callCost := call.CalculateCost()
		invoiceCalls = append(invoiceCalls, InvoiceCall{
			DestinationPhone: call.DestinationPhone,
			Duration:         call.Duration,
			Timestamp:        call.Date,
			Amount:           call.CalculateCost(),
		})

		totalAmount += callCost
		if callType == CallTypeFriend {
			totalFriendsSeconds += call.Duration
		}
		if callType == CallTypeInternational {
			totalInternationalSeconds += call.Duration
		}
		if callType == CallTypeNational {
			totalNationalSeconds += call.Duration
		}
	}

	return Invoice{
		User: InvoiceUser{
			Address: usr.Address,
			Name:    usr.Name,
			Phone:   string(usr.Phone),
		},
		Calls:                     invoiceCalls,
		TotalFriendsSeconds:       totalFriendsSeconds,
		TotalNationalSeconds:      totalNationalSeconds,
		TotalInternationalSeconds: totalInternationalSeconds,
		InvoiceTotal:              totalAmount,
	}, nil
}
