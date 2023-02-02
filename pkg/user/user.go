package user

import "errors"

// Nota: Este tipo solo lo declaro para que la definición de friends quede más
// declarativa, sino sería []string y ameritaría un comentario.
type PhoneNumber string

// User is a telephone line user
type User struct {
	Name    string
	Address string
	Phone   PhoneNumber
	Friends []PhoneNumber
}

// TODO: Considerar mover a caller
type Finder interface {
	FindByPhone(phoneNumber PhoneNumber) (User, error)
}

func FindByPhone(phoneNumber PhoneNumber) (User, error) {
	// TODO: POST a https://interview-brubank-api.herokuapp.com/users/:phoneNumber
	return User{}, errors.New("not implemented")
}

type MockFinder struct {
	users map[PhoneNumber]User
}

func NewMockFinder(users map[PhoneNumber]User) MockFinder {
	return MockFinder{}
}

func (m MockFinder) FindByPhone(phoneNumber PhoneNumber) (User, error) {
	usr, ok := m.users[phoneNumber]
	if !ok {
		return usr, errors.New("user not found")
	}

	return usr, nil
}
