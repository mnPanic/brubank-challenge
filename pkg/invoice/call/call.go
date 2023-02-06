// Package call implements calls and cost calculations based on their type and
// promotions. To process calls use the CallProcessor.
package call

import (
	"fmt"
	"invoice-generator/pkg/user"
	"regexp"
	"strings"
	"time"
)

type Call struct {
	DestinationPhone string
	SourcePhone      string
	Duration         uint // Seconds
	Date             time.Time
}

func New(destPhone string, sourcePhone string, duration uint, date time.Time) (Call, error) {
	if err := ValidatePhoneNumber(destPhone); err != nil {
		return Call{}, fmt.Errorf("destination phone: %s", err)
	}

	if err := ValidatePhoneNumber(sourcePhone); err != nil {
		return Call{}, fmt.Errorf("source phone: %s", err)
	}

	return Call{
		DestinationPhone: destPhone,
		SourcePhone:      sourcePhone,
		Duration:         duration,
		Date:             date,
	}, nil
}

// Phone number format: +549XXXXXXXXXX
// Nota de diseño: asumo que son válidos de 12 a 13 porque así están en los
// datos de ejemplo provistos.
var phoneNumberFormat = regexp.MustCompile(`\+[0-9]{12,13}`)

func ValidatePhoneNumber(phoneNumber string) error {
	if !phoneNumberFormat.MatchString(phoneNumber) {
		return fmt.Errorf("invalid format, should match %s", phoneNumberFormat.String())
	}

	return nil
}

// Type returns the type of the call
func (c Call) Type(friends []user.PhoneNumber) Type {
	if c.isFriend(friends) {
		return FriendCall{subtype: c.baseType()}
	}

	return c.baseType()
}

func (c Call) baseType() Type {
	sourceCountry := getCountryCode(c.SourcePhone)
	destinationCountry := getCountryCode(c.DestinationPhone)

	if sourceCountry == destinationCountry {
		return NationalCall{}
	}

	// Nota: Asumo que los códigos interplanetarios comienzan con 0
	isToOtherPlanet := strings.HasPrefix(destinationCountry, "0")
	if isToOtherPlanet {
		return InterplanetaryCall{durationSecs: c.Duration}
	}

	return InternationalCall{durationSecs: c.Duration}
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

// destinationCountryCode assumes that the phone number has a valid format,
//	+ (2 digit country) (11 digit number)
// For example,
//	+549XXXXXXXXXX -> 54
func getCountryCode(phoneNumber string) string {
	return phoneNumber[1:3]
}

// A Characteristic of a call, orthogonal to their type
type Characteristic uint

const (
	CharacteristicToFriend Characteristic = iota + 1
	CharacteristicInternational
)

type Type interface {
	BaseCost() float64
	RegisterDuration(uint, DurationRegisterer)
	HasCharacteristic(Characteristic) bool
}

// A DurationRegisterer knows how to register durations of different types of
// calls.
//
// Nota de diseño: Esta interfaz rara tuve que hacerla para evitar pasar un
// puntero a struct al CallType para que lo modifiquen, me parece que quedó un
// poco más limpio.
type DurationRegisterer interface {
	RegisterFriendCall(uint)
	RegisterNationalCall(uint)
	RegisterInternationalCall(uint)
	RegisterInterplanetaryCall(uint)
}

type InterplanetaryCall struct {
	durationSecs uint
}

func (c InterplanetaryCall) BaseCost() float64 {
	return float64(c.durationSecs) * 10
}

func (c InterplanetaryCall) HasCharacteristic(_ Characteristic) bool { return false }

func (c InterplanetaryCall) RegisterDuration(duration uint, registerer DurationRegisterer) {
	registerer.RegisterInterplanetaryCall(duration)
}

type InternationalCall struct {
	durationSecs uint
}

func (c InternationalCall) BaseCost() float64 {
	return float64(c.durationSecs)
}

func (c InternationalCall) HasCharacteristic(_ Characteristic) bool { return false }

func (c InternationalCall) RegisterDuration(duration uint, registerer DurationRegisterer) {
	registerer.RegisterInternationalCall(duration)
}

type NationalCall struct{}

func (c NationalCall) BaseCost() float64 {
	return 2.5
}

func (c NationalCall) HasCharacteristic(_ Characteristic) bool { return false }

func (c NationalCall) RegisterDuration(duration uint, registerer DurationRegisterer) {
	registerer.RegisterNationalCall(duration)
}

// FriendCall is a call characteristic
type FriendCall struct {
	subtype Type
}

func (c FriendCall) BaseCost() float64 {
	return c.subtype.BaseCost()
}

func (c FriendCall) HasCharacteristic(characteristic Characteristic) bool {
	return characteristic == CharacteristicToFriend
}

func (c FriendCall) RegisterDuration(duration uint, registerer DurationRegisterer) {
	registerer.RegisterFriendCall(duration)

	// Friend call register durations for friend and their base type
	c.subtype.RegisterDuration(duration, registerer)
}
