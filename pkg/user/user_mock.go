package user

import "errors"

// MockFinder is a mock Finder implementation over a Map.
type MockFinder struct {
	users map[PhoneNumber]User
}

// Verify interface compliance
var _ Finder = MockFinder{}

// NewMockFinderForUser returns a mock finder that can find the specified user
// by their phone.
//
// Nota de diseño: esta interfaz solo permite un usuario, que es lo que necesité
// en los tests. Claramente de ser necesario más de uno se podría extender.
func NewMockFinderForUser(usr User) MockFinder {
	return MockFinder{
		users: map[PhoneNumber]User{
			usr.Phone: usr,
		},
	}
}

func (m MockFinder) FindByPhone(phoneNumber PhoneNumber) (User, error) {
	usr, ok := m.users[phoneNumber]
	if !ok {
		return usr, errors.New("user not found")
	}

	return usr, nil
}
