package tokens

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/datastore"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/wind85/auth-api/core/config"
	"github.com/wind85/auth-api/core/errors"
	"github.com/wind85/auth-api/core/utils"
	"golang.org/x/net/xsrftoken"
)

var BlackList *tokenList

func init() {
	var err error
	BlackList, err = new(3, "Revoked Tokens")
	if err != nil {
		panic(err)
	}

}

type tokenList struct {
	Db   *datastore.Client
	tick *time.Ticker
	done chan bool
	kind string
}

type tokenElem struct {
	Token string
}

// new creates a new cache with minutes, which rappresent a interval at which
// old values are deleted and exp ( in minutes as well ) which sets the expiration
// time for values ( cannot be change once the cache has been istanciated ).
func new(minutes time.Duration, kind string) (*tokenList, error) {
	list := &tokenList{
		nil,
		time.NewTicker(time.Minute * minutes),
		make(chan bool, 1),
		kind,
	}

	var err error
	list.Db, err = datastore.NewClient(context.Background(), "boardsandwater")
	if err != nil {
		return nil, err
	}

	list.purger()

	return list, nil
}

// Put adds a values to the black list
func (c *tokenList) Put(jwt, claims string) error {
	_, err := c.Db.RunInTransaction(context.Background(),
		func(tx *datastore.Transaction) error {
			_, err := tx.Put(datastore.NameKey(c.kind, jwt, nil), &tokenElem{claims})
			if err != nil {
				return err
			}

			return nil
		})
	if err != nil {
		return err
	}

	return nil
}

// Get give you back the value assumining it hasn't be purged yet
func (c *tokenList) Valid(key string) bool {
	tok := &tokenElem{}

	err := c.Db.Get(context.Background(), datastore.NameKey(c.kind, key, nil), tok)
	if err == datastore.ErrNoSuchEntity {
		return true
	}

	// improvement needed here, it should not be better error checks
	return false
}

// Stop Must be called otherwise the cache will leak
func (c *tokenList) Stop() {
	c.done <- true
}

func (c *tokenList) purger() {
	go func() {
		for {
			select {
			case <-c.done:
				c.tick.Stop()
				return

			case <-c.tick.C:
				c.cleaner()
			}
		}
	}()
}

func (c *tokenList) cleaner() {
	iter := c.Db.Run(context.Background(), datastore.NewQuery("Revoked"))
	for {
		jwt := &tokenElem{}
		key, err := iter.Next(jwt)
		if err == iterator.Done {
			break
		}

		if err != nil {
			// log or do something
		}

		_, err = ClaimsFromJwt(jwt.Token)
		// verify proper error handling -- check docks
		if err == errors.NotValid {
			c.Db.Delete(context.Background(), key)
		}
	}
}

// jwt specifics

type customClaims struct {
	Custom string
	jwt.StandardClaims
}

func GenerateJwt(data []byte, delta int) string {
	mapped := string(data)

	claims := customClaims{
		mapped,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * time.Duration(delta)).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "waterandboards",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(utils.GetPrivateKey())
	if err != nil {
		return ""
	}

	return tokenString
}

func ClaimsFromJwt(tok string) (*customClaims, error) {

	token, err := jwt.ParseWithClaims(tok, &customClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, errors.WrongSigningMethod
			}
			// bolocks implementation http://stackoverflow.com/questions/28204385/using-jwt-go-library-key-is-invalid-or-invalid-type
			return utils.GetPubblicKey(), nil
		})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*customClaims)
	if !ok && !token.Valid {
		return nil, errors.NotValid
	}

	return claims, nil
}

type CrsfToken struct {
	Token string `json:"crsf"`
}

func GenerateCrsf(email string) ([]byte, error) {
	secret, err := config.Ini.GetString("crypto.secret")
	if err != nil {
		return nil, err
	}

	crsf_id, err := config.Ini.GetString("crypto.crsf_action_id")
	if err != nil {
		return nil, err
	}

	payload := xsrftoken.Generate(secret, email, crsf_id)
	csrf, err := json.Marshal(&CrsfToken{payload})
	if err != nil {
		return nil, err
	}

	return csrf, nil
}
