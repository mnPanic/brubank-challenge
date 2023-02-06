package call

import "invoice-generator/pkg/user"

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
	isCallToFriend := call.Type(p.usr.Friends).HasCharacteristic(CharacteristicToFriend)

	return isCallToFriend && didntExceedMax
}

func (p *promotionCallToFriends) Apply(call Call) float64 {
	p.currentFreeCallsToFriends++
	return 0
}

type promotionInternationalCallsToMercosur struct {
	usr user.User
}

func NewPromotionInternationalCallsToMercosur(usr user.User) promotionInternationalCallsToMercosur {
	return promotionInternationalCallsToMercosur{usr: usr}
}

func (p promotionInternationalCallsToMercosur) AppliesTo(call Call) bool {
	isInternational := call.Type(p.usr.Friends).HasCharacteristic(CharacteristicInternational)

	return isInternational && isMercosurPhone(call.DestinationPhone)
}

func isMercosurPhone(phoneNumber string) bool {
	mercosurCountryCodes := []string{"54", "60", "12"}

	phoneCountry := getCountryCode(phoneNumber)
	for _, code := range mercosurCountryCodes {
		if code == phoneCountry {
			return true
		}
	}

	return false
}

func (p promotionInternationalCallsToMercosur) Apply(call Call) float64 {
	// 50% off
	return call.Type(p.usr.Friends).BaseCost() * 0.5
}
