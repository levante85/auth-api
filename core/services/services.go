package services

import (
	"encoding/json"
	"log"

	"golang.org/x/crypto/bcrypt"

	"github.com/wind85/auth-api/core/config"
	"github.com/wind85/auth-api/core/errors"
	"github.com/wind85/auth-api/core/models"
	"github.com/wind85/auth-api/core/proxy"
	"github.com/wind85/auth-api/core/tokens"
	"github.com/wind85/auth-api/core/utils"
)

type Users struct {
	pool *proxy.Pool
}

func New(poolsize int) *Users {
	return &Users{
		proxy.NewPool(poolsize),
	}
}

func (u *Users) Login(email, password string) (string, []byte, error) {

	mng := u.pool.Get()
	defer u.pool.Put(mng)

	user, err := mng.Get(&models.User{Email: email, Password: password})
	if err != nil {
		return "", nil, err
	}

	err = utils.CheckPassword(user.Password, password)
	if err != nil {
		return "", nil, err
	}

	csrf, err := tokens.GenerateCrsf(user.Email)
	if err != nil {
		return "", nil, err
	}

	jwt_delta, err := config.Ini.GetInt("jwt_delta.delta")
	if err != nil {
		return "", nil, err
	}

	return tokens.GenerateJwt([]byte(user.Email), int(jwt_delta)), csrf, nil
}

func (u *Users) Register(data *models.User) error {
	mng := u.pool.Get()
	defer u.pool.Put(mng)

	user, err := mng.Create(data)
	if err != nil {
		return err
	}

	err = u.sendConfirmEmail(
		user.Email,
		"Registration Link",
		"registration",
		"activation/confirm", "",
	)
	if err != nil {
		return err
	}

	return nil
}

func (u *Users) Logout(jwt, claims string) error {
	if err := tokens.BlackList.Put(jwt, claims); err != nil {
		log.Println("Logout error: ", err)
	}

	return nil
}

func (u *Users) Me(email string, data *models.User) (*models.User, error) {
	mng := u.pool.Get()
	defer u.pool.Put(mng)

	if data != nil {
		user, err := mng.Update(data)
		if err != nil {
			return nil, err
		}

		return user, nil
	}

	other, err := mng.Get(&models.User{Email: email})
	if err != nil {
		return nil, err
	}

	return other, nil
}

func (u *Users) Activation(data *models.User) error {
	err := u.sendConfirmEmail(
		data.Email,
		"Activation Link",
		"activation_confirm",
		"activation/confirm", "",
	)
	if err != nil {
		return err
	}

	return nil
}

func (u *Users) ActivationConfirm(data []byte) error {
	mng := u.pool.Get()
	defer u.pool.Put(mng)

	claims, err := tokens.ClaimsFromJwt(string(data))
	if err != nil {
		return errors.New(err.Error())
	}

	gotUser, err := mng.Get(&models.User{Email: claims.Custom})
	if err != nil {
		return err
	}

	if gotUser.Code != string(data) {
		return errors.CodeNotValid
	}

	activate := &models.User{Isactive: "true", Email: claims.Custom}
	user, err := mng.Update(activate)
	if err != nil {
		return err
	}

	err = utils.SendEmail(
		[]string{user.Email},
		&utils.Email{"Activation Confirmed", ""},
		"activation_confirm",
	)
	if err != nil {
		return err
	}

	return nil
}

func (u *Users) PasswordReset(data *models.User) error {
	jwt_delta, err := config.Ini.GetInt("jwt_delta.delta")
	if err != nil {
		return err
	}

	code := tokens.GenerateJwt(nil, int(jwt_delta))

	//u.cache.Put(code, data)

	err = u.sendConfirmEmail(
		data.Email,
		"Password Reset Link",
		"password_reset",
		"password/reset/confirm",
		code,
	)
	if err != nil {
		return err
	}

	return nil
}

type passwordReset struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (u *Users) PasswordResetConfirm(data []byte) error {
	//value, ok := u.cache.Get(string(data))
	//if !ok {
	//		return errors.UserNotFound
	//	}

	//code, ok := value.([]byte)
	//if !ok {
	//		return errors.NotValid
	//	}

	content := &passwordReset{}
	// byte placeholder till I moved it to use another blacklist
	err := json.Unmarshal([]byte(""), content)
	if err != nil {
		return errors.JsonPayload
	}

	pass, err := bcrypt.GenerateFromPassword(
		[]byte(content.Password), bcrypt.MinCost,
	)
	if err != nil {
		return errors.InternalError
	}

	mng := u.pool.Get()
	defer u.pool.Put(mng)

	userup := &models.User{Email: content.Email, Password: string(pass)}

	user, err := mng.Update(userup)
	if err != nil {
		return err
	}

	err = utils.SendEmail(
		[]string{user.Email},
		&utils.Email{"password reset confirmed", ""},
		"password_reset_confirm",
	)
	if err != nil {
		return err
	}
	return nil
}

func (u *Users) sendConfirmEmail(email, title, tmplname, purl, code string) error {
	var url string

	mng := u.pool.Get()
	defer u.pool.Put(mng)

	user, err := mng.Get(&models.User{Email: email})
	if err != nil {
		return err
	}

	if code != "" {
		url = GenConfirmationUrl(user, purl, code)
	} else {
		url = GenConfirmationUrl(user, purl, user.Code)
	}

	err = utils.SendEmail(
		[]string{user.Email},
		&utils.Email{title, url},
		tmplname,
	)
	if err != nil {
		return err
	}

	return nil
}
