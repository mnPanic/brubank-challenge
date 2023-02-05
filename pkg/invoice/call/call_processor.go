package call

import (
	"invoice-generator/pkg/platform/timeutil"
	"invoice-generator/pkg/user"
)

// A Processor processes calls for a user one by one, returning their cost
// with any suitable promotions applied. It also summarizes their durations and
// total amount for statistical purposes.
type Processor struct {
	usr           user.User
	billingPeriod timeutil.Period
	promotions    []Promotion

	totalDurations TotalCallDurations
	totalAmount    float64
}

type TotalCallDurations struct {
	TotalInternationalSeconds  uint
	TotalNationalSeconds       uint
	TotalFriendsSeconds        uint
	TotalInterplanetarySeconds uint
}

// NewProcessor constructs a call processor.
func NewProcessor(usr user.User, period timeutil.Period, promotions []Promotion) Processor {
	return Processor{
		totalDurations: TotalCallDurations{},
		totalAmount:    0,

		usr:           usr,
		billingPeriod: period,
		promotions:    promotions,
	}
}

// Summarize returns the total amount and summarized durations of the calls.
func (c *Processor) Summarize() (float64, TotalCallDurations) {
	return c.totalAmount, c.totalDurations
}

// Process a call and return its cost. A call is skipped if it doesn't belong to
// the user we're processing or if it was made outside of the billing period.
func (c *Processor) Process(call Call) (cost float64, skip bool) {
	if c.shouldSkipCall(call) {
		return 0, true
	}

	callType := call.Type(c.usr.Friends)

	callType.RegisterDuration(call.Duration, c)

	callCost := c.callCost(call, callType)
	c.totalAmount += callCost
	return callCost, false
}

func (c *Processor) shouldSkipCall(call Call) bool {
	isOutsideBillingPeriod := !c.billingPeriod.Contains(call.Date)
	madeByOtherUser := string(c.usr.Phone) != call.SourcePhone

	return isOutsideBillingPeriod || madeByOtherUser
}

func (c *Processor) callCost(call Call, callType Type) float64 {
	for _, promo := range c.promotions {
		if promo.AppliesTo(call) {
			return promo.Apply(call)
		}
	}

	return callType.BaseCost()
}

// Methods to implement DurationRegisterer

func (c *Processor) RegisterFriendCall(duration uint) {
	c.totalDurations.TotalFriendsSeconds += duration
}

func (c *Processor) RegisterNationalCall(duration uint) {
	c.totalDurations.TotalNationalSeconds += duration
}

func (c *Processor) RegisterInternationalCall(duration uint) {
	c.totalDurations.TotalInternationalSeconds += duration
}

func (c *Processor) RegisterInterplanetaryCall(duration uint) {
	c.totalDurations.TotalInterplanetarySeconds += duration
}
