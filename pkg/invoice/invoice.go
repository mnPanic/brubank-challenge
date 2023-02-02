package invoice

import (
	"fmt"
	"invoice-generator/pkg/user"
)

type Invoice struct {
	User                      User `json:"user"`
	Calls                     []InvoiceCall
	TotalInternationalSeconds int `json:"total_international_seconds"`
	TotalNationalSeconds      int
	TotalFriendsSeconds       int
	InvoiceTotal              float64
}

type User struct {
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
		return fmt.Errorf("user not found, finder: %s", err)
	}
	return Invoice{}
}
