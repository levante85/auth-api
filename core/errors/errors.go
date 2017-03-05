package errors

import (
	"encoding/json"
	"errors"
)

var ErrMalformedInput = New("Malformed data!")
var ErrInternalDb = New("Internal Database problem!")
var ErrUserNotFound = New("User not found")
var ErrLoginError = New("Something is wrong in your credentials!")
var ErrNotValid = New("Your token autorization isn't valid!")
var ErrWrongSigningMethod = New("Wrong signing method!")
var ErrDontMatch = New("Your csrf token is not valid!")
var ErrBodyNotValid = New("Request body not valid!")
var ErrCrsfMissing = New("Crsf token is missing!")
var ErrTokCookieMissing = New("Jwt missing from cookie!")
var ErrInternalError = New("Internal Sever error!")
var ErrCookieNotFound = New("Error cookie not found")
var ErrCodeNotValid = New("Confirmation code don't match!")
var ErrNotBool = errors.New("Not a bool value")
var ErrNotString = errors.New("Not a string value")

func New(msg string) *ApiError {
	return &ApiError{msg}
}

type ApiError struct {
	Msg string `json:"error"`
}

func (a *ApiError) Error() string {
	return a.Msg
}

func Json(err error) []byte {
	a, _ := err.(*ApiError)
	b, _ := json.Marshal(a)

	return b
}
