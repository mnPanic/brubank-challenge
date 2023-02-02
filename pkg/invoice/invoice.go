package invoice

import (
	"fmt"
	"invoice-generator/pkg/user"
	"regexp"
	"time"
)

// Phone number format: +549XXXXXXXXXX
var phoneNumberFormat = regexp.MustCompile(`\+[0-9]{13}`)

// TODO: JSON tags (debería saltar en el test)
type Invoice struct {
	User                      InvoiceUser `json:"user"`
	Calls                     []InvoiceCall
	TotalInternationalSeconds uint `json:"total_international_seconds"`
	TotalNationalSeconds      uint
	TotalFriendsSeconds       uint
	InvoiceTotal              float64
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

type Call struct {
	DestinationPhone string
	SourcePhone      string
	Duration         uint
	Date             time.Time // ISO 8601 in UTC
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

// TODO: Mover a otro lado
// TimePeriod represents a period of time
type TimePeriod struct {
	Start time.Time
	End   time.Time
}

func (p TimePeriod) Contains(t time.Time) bool {
	return t.After(p.Start) && t.Before(p.End)
}

var timeLayoutISO8601 = "2006-01-02T15:04:05-0700"

// Generate generates an invoice for a given user with calls.
// It finds the user with the specified number (returning an error if it fails)
// and calculates the cost for each call.
func Generate(
	userFinder user.Finder,
	userPhoneNumber string,
	billingPeriod TimePeriod,
	calls []Call,
) (Invoice, error) {
	usr, err := userFinder.FindByPhone(user.PhoneNumber(userPhoneNumber))
	if err != nil {
		return Invoice{}, fmt.Errorf("finding user: %s", err)
	}

	// TODO: Falopa para que quede más limpio: mapa indexado por CallType que
	// tiene como valor un int.
	var (
		totalInternationalSeconds uint
		totalNationalSeconds      uint
		totalFriendsSeconds       uint

		totalAmount float64

		currentFriendCalls int // To charge after the 10th
	)

	// TODO: usar make para hacer más eficiente la transformación
	var invoiceCalls []InvoiceCall
	for i, call := range calls {

		if err := validateCall(call); err != nil {
			return Invoice{}, fmt.Errorf("invalid call #%d: %s", i, err)
		}

		if shouldSkipCall(call, userPhoneNumber, billingPeriod) {
			continue
		}

		// TODO: Evitar calcular dos veces el call type
		callType := call.Type(usr.Friends)
		callCost := call.CalculateCost(usr.Friends, currentFriendCalls)

		invoiceCalls = append(invoiceCalls, InvoiceCall{
			DestinationPhone: call.DestinationPhone,
			Duration:         call.Duration,
			Timestamp:        call.Date.Format(timeLayoutISO8601),
			Amount:           callCost,
		})

		totalAmount += callCost
		if callType.IsFriend {
			totalFriendsSeconds += call.Duration
			currentFriendCalls += 1
		}

		// Also count friend call duration for normal calls
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

func validateCall(call Call) error {
	if !phoneNumberFormat.MatchString(call.DestinationPhone) {
		return fmt.Errorf("invalid destination phone format, should match %s", phoneNumberFormat.String())
	}

	if !phoneNumberFormat.MatchString(call.SourcePhone) {
		return fmt.Errorf("invalid source phone format, should match %s", phoneNumberFormat.String())
	}

	return nil
}

func shouldSkipCall(call Call, userPhoneNumber string, billingPeriod TimePeriod) bool {
	isOutsideBillingPeriod := !billingPeriod.Contains(call.Date)
	madeByOtherUser := userPhoneNumber != call.SourcePhone

	return isOutsideBillingPeriod || madeByOtherUser
}
