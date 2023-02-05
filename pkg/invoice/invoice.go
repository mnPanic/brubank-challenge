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

type DurationRegisterer interface {
	RegisterFriendCall(uint)
	RegisterNationalCall(uint)
	RegisterInternationalCall(uint)
}

type CallType interface {
	BaseCost() float64
	RegisterDuration(uint, DurationRegisterer)
	HasCharacteristic(string) bool
}

type InternationalCall struct {
	durationSecs uint
}

func (c InternationalCall) BaseCost() float64 {
	return float64(c.durationSecs)
}

func (c InternationalCall) HasCharacteristic(_ string) bool { return false }

func (c InternationalCall) RegisterDuration(duration uint, registerer DurationRegisterer) {
	registerer.RegisterInternationalCall(duration)
}

type NationalCall struct{}

func (c NationalCall) BaseCost() float64 {
	return 2.5
}

func (c NationalCall) HasCharacteristic(_ string) bool { return false }

func (c NationalCall) RegisterDuration(duration uint, registerer DurationRegisterer) {
	registerer.RegisterNationalCall(duration)
}

// FriendCall is a composite call type
type FriendCall struct {
	subtype CallType
}

func (c FriendCall) BaseCost() float64 {
	return c.subtype.BaseCost()
}

const (
	callCharacteristicToFriend = "call_to_friend"
)

func (c FriendCall) HasCharacteristic(characteristic string) bool {
	return characteristic == callCharacteristicToFriend
}

func (c FriendCall) RegisterDuration(duration uint, registerer DurationRegisterer) {
	registerer.RegisterFriendCall(duration)
	// Friend call register durations for friend and their base type
	c.subtype.RegisterDuration(duration, registerer)
}

func (c Call) Type(friends []user.PhoneNumber) CallType {
	if c.isFriend(friends) {
		return FriendCall{subtype: c.baseType()}
	}

	return c.baseType()
}

func (c Call) baseType() CallType {
	if c.isNational() {
		return NationalCall{}
	}

	return InternationalCall{durationSecs: c.Duration}
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

type Promotion interface {
	// AppliesTo returns whether a promotion applies to a call
	AppliesTo(Call) bool

	// Apply applies the promotion to the call, returning the final cost
	Apply(Call) float64
}

type promotionCallToFriends struct {
	usr user.User

	currentFreeCallsToFriends uint
}

func NewPromotionFreeCallsToFriends(usr user.User) *promotionCallToFriends {
	return &promotionCallToFriends{
		usr:                       usr,
		currentFreeCallsToFriends: 0,
	}
}

func (p *promotionCallToFriends) AppliesTo(call Call) bool {
	const maxFreeCallsToFriends = 10
	didntExceedMax := p.currentFreeCallsToFriends < maxFreeCallsToFriends
	isCallToFriend := call.Type(p.usr.Friends).HasCharacteristic(callCharacteristicToFriend)

	return isCallToFriend && didntExceedMax
}

func (p *promotionCallToFriends) Apply(call Call) float64 {
	p.currentFreeCallsToFriends++
	return 0
}

var errSkipCall = errors.New("skip call")

type totalSeconds struct {
	totalInternationalSeconds uint
	totalNationalSeconds      uint
	totalFriendsSeconds       uint
}

type callProcessor struct {
	usr           user.User
	billingPeriod timeutil.Period
	promotions    []Promotion

	seconds     totalSeconds
	totalAmount float64
}

func NewCallProcessor(usr user.User, period timeutil.Period, promotions []Promotion) callProcessor {
	return callProcessor{
		seconds:     totalSeconds{},
		totalAmount: 0,

		usr:           usr,
		billingPeriod: period,
		promotions:    promotions,
	}
}

func (c *callProcessor) finish() (float64, totalSeconds) {
	return c.totalAmount, c.seconds
}

func (c *callProcessor) process(call Call) (float64, error) {
	if c.shouldSkipCall(call) {
		return 0, errSkipCall
	}

	callType := call.Type(c.usr.Friends)

	callType.RegisterDuration(call.Duration, c)

	callCost := c.callCost(call, callType)
	c.totalAmount += callCost
	return callCost, nil
}

func (c *callProcessor) shouldSkipCall(call Call) bool {
	isOutsideBillingPeriod := !c.billingPeriod.Contains(call.Date)
	madeByOtherUser := string(c.usr.Phone) != call.SourcePhone

	return isOutsideBillingPeriod || madeByOtherUser
}

func (c *callProcessor) callCost(call Call, callType CallType) float64 {
	for _, promo := range c.promotions {
		if promo.AppliesTo(call) {
			return promo.Apply(call)
		}
	}

	return callType.BaseCost()
}

func (c *callProcessor) RegisterFriendCall(duration uint) {
	c.seconds.totalFriendsSeconds += duration
}

func (c *callProcessor) RegisterNationalCall(duration uint) {
	c.seconds.totalNationalSeconds += duration
}

func (c *callProcessor) RegisterInternationalCall(duration uint) {
	c.seconds.totalInternationalSeconds += duration
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

	callProcessor := NewCallProcessor(usr, billingPeriod, []Promotion{
		NewPromotionFreeCallsToFriends(usr),
	})

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
