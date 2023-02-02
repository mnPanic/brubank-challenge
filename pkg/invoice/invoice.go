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

// TODO. Idea de abstracción: cost calculator instanciado antes de cada una.
// Internamente guarda el contador de las llamadas a amigos.

// TODO: Tiene sentido que esto sea parte de call, o un modulo a parte
func (c Call) CalculateCost(friends []user.PhoneNumber, currentFriendCalls int) float64 {
	const maxFreeFriendCalls = 10
	callType := c.Type(friends)

	if callType.IsFriend && currentFriendCalls < maxFreeFriendCalls {
		return 0
	}

	// Let it through and charge it as a call to a stranger (either national or
	// international)

	if callType.IsNational {
		return 2.5
	}

	// International
	return float64(c.Duration)
}

type CallType struct {
	IsNational bool // National or international
	IsFriend   bool // Friend or stranger
}

func (c Call) Type(friends []user.PhoneNumber) CallType {
	return CallType{
		IsNational: c.isNational(),
		IsFriend:   c.isFriend(friends),
	}
}

// isNational returns whether the call was made to the same country (by
// comparing source and destination country codes)
func (c Call) isNational() bool {
	sourceCountry := getCountryCode(c.SourcePhone)
	destinationCountry := getCountryCode(c.DestinationPhone)

	return sourceCountry == destinationCountry
}

// destinationCountryCode assumes that the phone number has a valid format,
//	+ (2 digit country) (11 digit number)
// For example,
//	+549XXXXXXXXXX -> 54
func getCountryCode(phoneNumber string) string {
	return phoneNumber[1:3]
}

// isFriend returns whether this call was made to a friend
func (c Call) isFriend(friends []user.PhoneNumber) bool {
	for _, friendPhone := range friends {
		if c.DestinationPhone == string(friendPhone) {
			return true
		}
	}

	return false
}

// Generate generates an invoice for a given user with calls.
// It finds the user with the specified number (returning an error if it fails)
// and calculates the cost for each call.
func Generate(
	userFinder user.Finder,
	userPhoneNumber string,
	billingStartDate string,
	billingEndDate string,
	calls []Call,
) (Invoice, error) {
	usr, err := userFinder.FindByPhone(user.PhoneNumber(userPhoneNumber))
	if err != nil {
		return Invoice{}, fmt.Errorf("finding user: %s", err)
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

		currentFriendCalls int // To charge after the 10th
	)

	// TODO: usar make para hacer más eficiente la transformación
	var invoiceCalls []InvoiceCall
	for _, call := range calls {
		// TODO: Evitar calcular dos veces el call type
		callType := call.Type(usr.Friends)
		callCost := call.CalculateCost(usr.Friends, currentFriendCalls)
		invoiceCalls = append(invoiceCalls, InvoiceCall{
			DestinationPhone: call.DestinationPhone,
			Duration:         call.Duration,
			Timestamp:        call.Date,
			Amount:           callCost,
		})

		totalAmount += callCost
		if callType.IsFriend {
			totalFriendsSeconds += call.Duration
			currentFriendCalls += 1
		}

		// Also count friend calls as normal calls
		if callType.IsNational {
			totalNationalSeconds += call.Duration
		} else {
			totalInternationalSeconds += call.Duration
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
