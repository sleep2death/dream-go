package dream

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

type wxAuth struct {
	OpenId     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionId    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func authHandlers() {
	r.POST("/api/auth/login", loginHandler)
	r.POST("/api/auth/signup", signupHandler)
	r.GET("/api/auth/wx/:code", wxLoginHandler)
	r.GET("/api/auth/wx_token/:token", wxTokenHandler)
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

func wxJwtAuth(c *gin.Context) {
	str, err := c.Cookie("jwt_token")
	if err != nil {
		permissionError(c, errAuthFailed)
		c.Abort()
		return
	}

	var cl jwt.MapClaims
	token, err := jwt.ParseWithClaims(str, &cl, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("jwt_key")), nil
	})

	if err != nil {
		permissionError(c, errAuthFailed)
		c.Abort()
		return
	}

	if token.Valid {
		// log.Printf("%v %v", claims.Email, claims.StandardClaims.Issuer)
		c.Set("id", cl["id"])
		c.Set("session", cl["session"])
		c.Next()
	} else {
		permissionError(c, errAuthFailed)
		c.Abort()
	}
}

func wxLoginHandler(c *gin.Context) {
	code := c.Param("code")

	if len(code) == 0 {
		badRequest(c, errors.New("error.empty_param"))
		return
	}

	appSecret := viper.GetString("WECHAT_SECRET")
	if len(code) == 0 {
		internalError(c, errors.New("wechat app-secret not found"))
		return
	}

	appId := viper.GetString("WECHAT_APP")
	if len(code) == 0 {
		internalError(c, errors.New("wechat app-id not found"))
		return
	}

	appGrant := viper.GetString("WECHAT_GRANT")
	if len(code) == 1 {
		internalError(c, errors.New("wechat app-grant-type not found"))
		return
	}

	resp, err := http.Get(fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=%s", appId, appSecret, code, appGrant))
	if err != nil {
		internalError(c, err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		internalError(c, err)
		return
	}

	var result wxAuth
	err = json.Unmarshal(body, &result)
	if err != nil {
		internalError(c, err)
		return
	}

	if result.ErrCode != 0 {
		badRequest(c, errors.New(result.ErrMsg))
		return
	}

	id := "wx:" + result.OpenId

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":      id,
		"session": result.SessionKey,
	})
	ss, err := token.SignedString([]byte(viper.GetString("jwt_key")))
	if err != nil {
		internalError(c, err)
	}

	_, err = getUserById(id)
	if err == mongo.ErrNoDocuments {
		l.Debugln("incoming new wechat user", result.OpenId)

		err = newWxUser("wx:" + result.OpenId)

		if err != nil {
			internalError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":    true,
			"isNew": true,
			"token": ss,
		})
	} else if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"isNew": false,
		"token": ss,
	})
}

func wxTokenHandler(c *gin.Context) {
	str, err := c.Cookie("jwt_token")
	if err != nil {
		permissionError(c, errAuthFailed)
		return
	}

	token, err := jwt.ParseWithClaims(str, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("jwt_key")), nil
	})

	if err != nil || !token.Valid {
		permissionError(c, errAuthFailed)
		return
	}

	ok(c)
}
