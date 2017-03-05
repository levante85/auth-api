package views

import (
	"log"
	"net/http"

	"github.com/auth-api/core/cookies"
	"github.com/auth-api/core/errors"
	"github.com/auth-api/core/models"
	"github.com/auth-api/core/services"
)

func Login(w http.ResponseWriter, r *http.Request) {
	data := ViewsModifierHelper(w, r)
	if data == nil {
		return
	}

	token, crsf, err := services.Login(data)
	if err != nil {
		HttpJsonError(w, err, http.StatusForbidden)
		return
	}

	cookies.Set(w, token)

	n, err := w.Write(crsf)
	if err != nil || n != len(crsf) {
		HttpJsonError(w, errors.ErrInternalError, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	token, crsf := GetCookieAndCrsf(w, r)

	if crsf != "" && token != "" {
		err := services.Logout(token, crsf)
		if err != nil {
			HttpJsonError(w, err, http.StatusNotAcceptable)
			return
		}

		cookies.Clear(w)

		w.WriteHeader(http.StatusOK)
		return
	}

	if crsf != "" && token == "" {
		HttpJsonError(w, errors.ErrCrsfMissing, http.StatusForbidden)
		return
	}

	if crsf == "" && token != "" {
		HttpJsonError(w, errors.ErrTokCookieMissing, http.StatusForbidden)
		return
	}
}

func Me(w http.ResponseWriter, r *http.Request) {
	token, crsf := GetCookieAndCrsf(w, r)
	if token == "" || crsf == "" {
		return
	}

	user := &models.User{}
	var err error

	if r.Method == http.MethodPut {
		data := ViewsModifierHelper(w, r)
		if data != nil {
			return
		}

		user, err = services.Me(token, crsf, data)
		if err != nil {
			HttpJsonError(w, err, http.StatusExpectationFailed)
			return
		}
	}

	HeaderHelper(w)

	if r.Method == http.MethodGet {
		user, err = services.Me(token, crsf, nil)
		if err != nil {
			log.Println("Me get view error")
			HttpJsonError(w, err, http.StatusExpectationFailed)
			return

		}
	}

	w.Write(Serialize(user))
	w.WriteHeader(http.StatusOK)
}

func Register(w http.ResponseWriter, r *http.Request) {
	data := ViewsModifierHelper(w, r)
	if data != nil {
		err := services.Registration(data)
		if err != nil {
			HttpJsonError(w, err, http.StatusExpectationFailed)
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func Activate(w http.ResponseWriter, r *http.Request) {
	data := ViewsModifierHelper(w, r)
	if data != nil {
	}

	w.WriteHeader(http.StatusNotImplemented)
	return
}

func PasswordReset(w http.ResponseWriter, r *http.Request) {
	data := ViewsModifierHelper(w, r)
	if data != nil {
	}

	w.WriteHeader(http.StatusNotImplemented)
	return
}

func PasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
	data := ViewsModifierHelper(w, r)
	if data != nil {
	}

	w.WriteHeader(http.StatusNotImplemented)
	return
}
