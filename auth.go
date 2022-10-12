package dream

import (
	"errors"
	"net/http"
	"net/mail"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/viper"
	pwdValidator "github.com/wagslane/go-password-validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// var pwdReg = regexp2.MustCompile("^(?=.*[0-9])(?=.*[!@#$%^&*])[a-zA-Z0-9!@#$%^&*]{6,24}", 0)
var nameReg = regexp2.MustCompile("^[A-Za-z0-9_\u3000\u3400-\u4DBF\u4E00-\u9FFF]{6,24}$", 0)

var errLoginFailed = errors.New("login.failed")
var errAuthFailed = errors.New("auth.failed")

type claims struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func authHandlers() {
	r.POST("/api/auth/login", loginHandler)
	r.POST("/api/auth/signup", signupHandler)
}

func loginHandler(c *gin.Context) {
	id := c.PostForm("id")
	password := c.PostForm("password")

	// email validation
	_, err := mail.ParseAddress(id)
	isUsername := err != nil

	// password validation
	err = pwdValidator.Validate(password, viper.GetViper().GetFloat64("pwdMinStr"))
	if err != nil {
		badRequest(c, errLoginFailed)
		return
	}

	var res bson.M

	if isUsername {
		// username validation
		if match, err := nameReg.MatchString(id); err != nil || !match {
			badRequest(c, errLoginFailed)
			return
		}
		res, err = findUsrByName(id, nil)
	} else {
		// email
		res, err = findUsrByEmail(id, nil)
	}

	if err != nil {
		badRequest(c, errLoginFailed)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(res["password"].(string)), []byte(password))
	if err != nil {
		badRequest(c, errLoginFailed)
		return
	}

	expireTime := time.Now().Add(604800 * time.Second) // expires in seven days

	claims := claims{
		res["_id"].(string),
		res["username"].(string),
		res["email"].(string),
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime), // 7 days
			Issuer:    "minish-cap.com",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(viper.GetString("jwt_key")))
	if err != nil {
		internalError(c, err)
	}

	// set cookie with jwt token
	c.SetCookie("jwt_token", ss, 604800, "/", "", false, true) // set secure param to "false" for http request
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
	})

}

func signupHandler(c *gin.Context) {
	email := c.PostForm("email")
	username := c.PostForm("username")
	password := c.PostForm("password")

	// email validation
	if _, err := mail.ParseAddress(email); err != nil {
		badRequest(c, errors.New("signup.invalid_email"))
		return
	}

	// username validation
	if match, err := nameReg.MatchString(username); err != nil || !match {
		badRequest(c, errors.New("signup.invalid_username"))
		return
	}

	// password validation
	err := pwdValidator.Validate(password, viper.GetViper().GetFloat64("pwdMinStr"))
	if err != nil {
		badRequest(c, err)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		internalError(c, err)
		return
	}

	// finally, create the account
	err = newUser(email, username, string(hashedPassword))

	if err != nil {
		// if already exist
		if _, ok := err.(mongo.WriteException); ok {
			badRequest(c, errors.New("signup.duplicated"))
			return
		}

		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
	})
}

func jwtAuth(c *gin.Context) {
	str, err := c.Cookie("jwt_token")
	if err != nil {
		permissionError(c, errAuthFailed)
		c.Abort()
		return
	}

	token, err := jwt.ParseWithClaims(str, &claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("jwt_key")), nil
	})

	if err != nil {
		permissionError(c, errAuthFailed)
		c.Abort()
		return
	}

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		// log.Printf("%v %v", claims.Email, claims.StandardClaims.Issuer)

		c.Set("email", claims.Email)
		c.Set("username", claims.Name)
		c.Set("uuid", claims.ID)

		c.Next()
	} else {
		permissionError(c, errAuthFailed)
		c.Abort()
	}
}
