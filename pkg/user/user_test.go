package user_test

import (
	"encoding/json"
	"errors"
	"invoice-generator/pkg/user"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnServiceOKReturnsUser(t *testing.T) {
	finder := user.NewFinder(staticGetter{
		content: json.RawMessage(`{
			"name": "Hosea Nitzsche",
			"address": "77826 Jaime Mews",
			"phone_number": "+5491167980952",
			"friends": ["+5491167980953", "+5491167980951", "+191167980953"]
		}`),
		statusCode: http.StatusOK,
	})

	usr, err := finder.FindByPhone("+5491167980952")
	require.NoError(t, err)

	expectedUser := user.User{
		Address: "77826 Jaime Mews",
		Name:    "Hosea Nitzsche",
		Phone:   "+5491167980952",
		Friends: []user.PhoneNumber{"+5491167980953", "+5491167980951", "+191167980953"},
	}
	assert.Equal(t, expectedUser, usr)
}

func TestOnServiceUnexpectedStatusCodeReturnsError(t *testing.T) {
	finder := user.NewFinder(staticGetter{
		content:    nil,
		statusCode: http.StatusInternalServerError,
	})

	_, err := finder.FindByPhone("+5491167980952")
	require.EqualError(t, err, "unexpected status code (500) expected 200 OK")
}

func TestOnServiceInvalidJSONResponseReturnsError(t *testing.T) {
	finder := user.NewFinder(staticGetter{
		// Because name is an int, it can't be unmarshalled into a string and
		// will return an error
		content: json.RawMessage(`{
			"name": 11,
			"address": "77826 Jaime Mews",
			"friends": ["+5491167980953", "+5491167980951", "+191167980953"],
			"phone_number": "+5491167980952"
		}`),
		statusCode: http.StatusOK,
	})

	_, err := finder.FindByPhone("+5491167980952")
	require.EqualError(t, err, "parsing body: json: cannot unmarshal number into Go struct field User.name of type string")
}

func TestGetError(t *testing.T) {
	finder := user.NewFinder(staticGetter{err: errors.New("timeout")})

	_, err := finder.FindByPhone("+5491167980952")
	require.EqualError(t, err, "http get: timeout")
}

type staticGetter struct {
	content    json.RawMessage
	statusCode int

	err error
}

func (s staticGetter) Get(url string) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}

	rec := httptest.NewRecorder()
	if s.content != nil {
		rec.Write(s.content)
	}
	rec.WriteHeader(s.statusCode)
	return rec.Result(), nil
}
