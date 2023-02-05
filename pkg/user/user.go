package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Nota: Este tipo solo lo declaro para que la definición de friends quede más
// declarativa, sino sería []string y ameritaría un comentario.
type PhoneNumber string

// User is a telephone line user
type User struct {
	Name    string        `json:"name"`
	Address string        `json:"address"`
	Phone   PhoneNumber   `json:"phone_number"`
	Friends []PhoneNumber `json:"friends"`
}

// TODO: Considerar mover a caller
type Finder interface {
	FindByPhone(phoneNumber PhoneNumber) (User, error)
}

// HTTPGetter represents the types that know how to perform http GETs.
//
// Nota de diseño: Si se quisiera hacer más genérico se podría tener Do() en la
// interfaz.
type HTTPGetter interface {
	Get(url string) (*http.Response, error)
}

type UserFinder struct {
	getter HTTPGetter
}

func NewFinder(getter HTTPGetter) UserFinder {
	return UserFinder{getter: getter}
}

func (u UserFinder) FindByPhone(phoneNumber PhoneNumber) (User, error) {
	url := fmt.Sprintf("https://fn-interview-api.azurewebsites.net/users/%s", phoneNumber)
	resp, err := u.getter.Get(url)
	if err != nil {
		return User{}, fmt.Errorf("http get: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("unexpected status code (%d) expected 200 OK", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return User{}, fmt.Errorf("reading body: %s", err)
	}

	defer resp.Body.Close()

	var usr User
	err = json.Unmarshal(body, &usr)
	if err != nil {
		return User{}, fmt.Errorf("parsing body: %s", err)
	}

	if usr.Phone != phoneNumber {
		return User{}, errors.New("invalid response, phone numbers differ")
	}

	return usr, nil
}
