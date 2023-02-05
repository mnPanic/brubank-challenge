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
