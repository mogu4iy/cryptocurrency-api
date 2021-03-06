package controllers

import (
	"cryptocurrencies-api/config"
	"cryptocurrencies-api/models"
	u "cryptocurrencies-api/utils"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/schema"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

var decoder = schema.NewDecoder()

var CreateAccount = func(w http.ResponseWriter, r *http.Request) {
	account := &models.AccountSchema{}

	err := decoder.Decode(account, r.URL.Query())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		u.Respond(w, config.ControllersConfig.Messages["BadRequest"])
		return
	}

	selected, err := models.AccountStorage.SelectValues([]interface{}{account}, []string{"Email"})
	if err != nil{
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, config.ControllersConfig.Messages["InternalServerError"])
		return
	}
	if len(selected) > 0 {
		fmt.Println(selected)
		w.WriteHeader(http.StatusMethodNotAllowed)
		u.Respond(w, config.ControllersConfig.Messages["AccountExists"])
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(account.Password), bcrypt.DefaultCost)
	account.Password = string(hashedPassword)

	insertedUser, err := models.AccountStorage.InsertValues([]interface{}{account})
	if len(insertedUser) == 0 {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, config.ControllersConfig.Messages["InternalServerError"])
		return
	}

	accessToken, err := GenerateToken(account.Id, config.JwtConfig.AccessTokenExpiresMinutes)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, config.ControllersConfig.Messages["InternalServerError"])
		return
	}

	refreshToken, err := GenerateToken(account.Id, config.JwtConfig.RefreshTokenExpiresMinutes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, config.ControllersConfig.Messages["InternalServerError"])
		return
	}

	tokenUser := &models.TokenSchema{
		UserId: account.Id,
		RefreshToken: refreshToken,
	}

	insertedToken, err := models.TokenStorage.InsertValues([]interface{}{tokenUser})
	if len(insertedToken) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, config.ControllersConfig.Messages["InternalServerError"])
		return
	}

	response := config.ControllersConfig.Messages["AccountCreated"]
	response["accessToken"] = accessToken
	response["refreshToken"] = refreshToken

	u.Respond(w, response)
}

var LoginAccount = func(w http.ResponseWriter, r *http.Request) {
	account := &models.AccountSchema{}

	err := decoder.Decode(account, r.URL.Query())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		u.Respond(w, config.ControllersConfig.Messages["BadRequest"])
		return
	}

	selected, err := models.AccountStorage.SelectValues([]interface{}{account}, []string{"Email"})
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, config.ControllersConfig.Messages["InternalServerError"])
	}
	if len(selected) == 0 {
		fmt.Println(selected)
		w.WriteHeader(http.StatusMethodNotAllowed)
		u.Respond(w, config.ControllersConfig.Messages["InvalidEmailOrPassword"])
		return
	}

	password := u.GetField(selected, "Password").(string)

	err = bcrypt.CompareHashAndPassword([]byte(password), []byte(account.Password))
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword {
		w.WriteHeader(http.StatusForbidden)
		u.Respond(w,config.ControllersConfig.Messages["InvalidPassword"])
		return
	}

	account.Id = r.Context().Value("UserId")

	accessToken, err := GenerateToken(account.Id, config.JwtConfig.AccessTokenExpiresMinutes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, config.ControllersConfig.Messages["InternalServerError"])
		return
	}

	response := config.ControllersConfig.Messages["AOK"]
	response["accessToken"] = accessToken

	u.Respond(w, response)
}

var RefreshToken = func(w http.ResponseWriter, r *http.Request) {

	userId := r.Context().Value("UserId")

	accessToken, err := GenerateToken(userId, config.JwtConfig.AccessTokenExpiresMinutes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, u.Message(false, "Internal Server Error"))
		return
	}

	refreshToken, err :=  GenerateToken(userId, config.JwtConfig.RefreshTokenExpiresMinutes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, u.Message(false, "Internal Server Error"))
		return
	}

	userToken := models.TokenSchema{UserId: r.Context().Value("UserId")}
	if _, err := models.TokenStorage.UpdateValues([]interface{}{&userToken}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, u.Message(false, "Internal Server Error"))
		return
	}

	response := config.ControllersConfig.Messages["AccountHasBeenCreated"]
	response["accessToken"] = accessToken
	response["refreshToken"] = refreshToken

	u.Respond(w, response)
}

func GenerateToken(userId interface{}, expireMinutes uint) (string, error) {
	exp := time.Now().Add(time.Minute * time.Duration(expireMinutes)).Unix()

	tokenClaims := &config.TokenClaims{
		UserId: userId,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp,
		},
	}

	return jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tokenClaims).SignedString([]byte(config.JwtConfig.Secret))
}
