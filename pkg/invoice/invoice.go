package invoice

import (
	"errors"
	"fmt"
	"invoice-generator/pkg/platform/timeutil"
	"invoice-generator/pkg/user"
	"regexp"
	"time"
)

// Phone number format: +549XXXXXXXXXX
// Nota de diseño: asumo que son válidos de 12 a 13 porque así están en los
// datos de ejemplo provistos.
var phoneNumberFormat = regexp.MustCompile(`\+[0-9]{12,13}`)

type Invoice struct {
	User                      InvoiceUser   `json:"user"`
	Calls                     []InvoiceCall `json:"calls"`
	TotalInternationalSeconds uint          `json:"total_international_seconds"`
	TotalNationalSeconds      uint          `json:"total_national_seconds"`
	TotalFriendsSeconds       uint          `json:"total_friends_seconds"`
	InvoiceTotal              float64       `json:"total"`
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
	Duration         uint // Seconds
	Date             time.Time
}

func NewCall(destPhone string, sourcePhone string, duration uint, date time.Time) (Call, error) {
	if err := validatePhoneNumber(destPhone); err != nil {
		return Call{}, fmt.Errorf("destination phone: %s", err)
	}

	if err := validatePhoneNumber(sourcePhone); err != nil {
		return Call{}, fmt.Errorf("source phone: %s", err)
	}

	return Call{
		DestinationPhone: destPhone,
		SourcePhone:      sourcePhone,
		Duration:         duration,
		Date:             date,
	}, nil
}

func validatePhoneNumber(phoneNumber string) error {
	if !phoneNumberFormat.MatchString(phoneNumber) {
		return fmt.Errorf("invalid format, should match %s", phoneNumberFormat.String())
	}

	return nil
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

var errSkipCall = errors.New("skip call")

type totalSeconds struct {
	totalInternationalSeconds uint
	totalNationalSeconds      uint
	totalFriendsSeconds       uint
}

type callProcessor struct {
	seconds            totalSeconds
	totalAmount        float64
	currentFriendCalls int

	usr           user.User
	billingPeriod timeutil.Period
}

func NewCallProcessor(usr user.User, period timeutil.Period) callProcessor {
	return callProcessor{
		seconds:            totalSeconds{},
		totalAmount:        0,
		currentFriendCalls: 0,

		usr:           usr,
		billingPeriod: period,
	}
}

func (c *callProcessor) finish() (float64, totalSeconds) {
	return c.totalAmount, c.seconds
}

func (c *callProcessor) process(call Call) (float64, error) {
	if shouldSkipCall(call, string(c.usr.Phone), c.billingPeriod) {
		return 0, errSkipCall
	}

	// TODO: Evitar calcular dos veces el call type
	callType := call.Type(c.usr.Friends)
	callCost := call.CalculateCost(c.usr.Friends, c.currentFriendCalls)

	c.totalAmount += callCost
	if callType.IsFriend {
		c.seconds.totalFriendsSeconds += call.Duration
		c.currentFriendCalls += 1
	}

	// Also count friend call duration for normal calls
	if callType.IsNational {
		c.seconds.totalNationalSeconds += call.Duration
	} else {
		c.seconds.totalInternationalSeconds += call.Duration
	}

	return callCost, nil
}

// Generate generates an invoice for a given user with calls.
// It finds the user with the specified number (returning an error if it fails)
// and calculates the cost for each call.
func Generate(
	userFinder user.Finder,
	userPhoneNumber string,
	billingPeriod timeutil.Period,
	calls []Call,
) (Invoice, error) {
	usr, err := userFinder.FindByPhone(user.PhoneNumber(userPhoneNumber))
	if err != nil {
		return Invoice{}, fmt.Errorf("finding user: %s", err)
	}

	callProcessor := NewCallProcessor(usr, billingPeriod)

	var invoiceCalls []InvoiceCall
	for i, call := range calls {
		callCost, err := callProcessor.process(call)
		if errors.Is(err, errSkipCall) {
			continue
		}

		if err != nil {
			return Invoice{}, fmt.Errorf("processing call #%d: %s", i, err)
		}

		invoiceCalls = append(invoiceCalls, InvoiceCall{
			DestinationPhone: call.DestinationPhone,
			Duration:         call.Duration,
			Timestamp:        call.Date.Format(timeutil.LayoutISO8601),
			Amount:           callCost,
		})
	}

	totalAmount, totalSeconds := callProcessor.finish()

	return Invoice{
		User: InvoiceUser{
			Address: usr.Address,
			Name:    usr.Name,
			Phone:   string(usr.Phone),
		},
		Calls:                     invoiceCalls,
		TotalFriendsSeconds:       totalSeconds.totalFriendsSeconds,
		TotalNationalSeconds:      totalSeconds.totalNationalSeconds,
		TotalInternationalSeconds: totalSeconds.totalInternationalSeconds,
		InvoiceTotal:              totalAmount,
	}, nil
}

func shouldSkipCall(call Call, userPhoneNumber string, billingPeriod timeutil.Period) bool {
	isOutsideBillingPeriod := !billingPeriod.Contains(call.Date)
	madeByOtherUser := userPhoneNumber != call.SourcePhone

	return isOutsideBillingPeriod || madeByOtherUser
}
