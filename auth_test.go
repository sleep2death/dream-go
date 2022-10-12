package dream

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func testLogin(t *testing.T, name string) (w *httptest.ResponseRecorder) {
	req, err := postFormReq("/api/auth/signup", map[string]string{
		"username": name,
		"email":    name + "@test.com",
		"password": "Passw0rd!",
	})
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertOK(t, w)

	// login normally
	req, err = postFormReq("/api/auth/login", map[string]string{
		"id":       name,
		"password": "Passw0rd!",
	})
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return
}

func testJwtToken(t *testing.T, w *httptest.ResponseRecorder) (token *http.Cookie, c *claims) {
	// check jwt_token in the cookies
	for _, c := range w.Result().Cookies() {
		if c.Name == "jwt_token" {
			token = c
		}
	}
	assert.NotNil(t, token)

	// l.Infoln("jwt_token", token.Value, "jwt_key", viper.GetString("jwt_key"))
	// parse token
	jt, err := jwt.ParseWithClaims(token.Value, &claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("jwt_key")), nil
	})
	assert.Nil(t, err)
	c, ok := jt.Claims.(*claims)
	assert.True(t, ok)

	return
}

func TestSignupHandler(t *testing.T) {
	testSetup()

	defer func() {
		err := delUsrByName("tester002")
		if err != nil {
			t.Fatal(err)
		}
	}()

	req, err := postFormReq("/api/auth/signup", map[string]string{
		"username": "tester002",
		"email":    "tester002@test.com",
		"password": "Passw0rd!",
	})
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertOK(t, w)

	// invalid username
	req, err = postFormReq("/api/auth/signup", map[string]string{
		"username": "te", // invalid username
		"email":    "tester002@test.com",
		"password": "Passw0rd!",
	})

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := assertNotOK(t, w)
	assert.Equal(t, "signup.invalid_username", body["msg"])

	// empty or null username
	req, err = postFormReq("/api/auth/signup", map[string]string{
		"email":    "tester002@test.com",
		"password": "Passw0rd!",
	})

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body = assertNotOK(t, w)
	assert.Equal(t, "signup.invalid_username", body["msg"])

	// invalid email
	req, err = postFormReq("/api/auth/signup", map[string]string{
		"username": "tester002",
		"email":    "tester002@.com",
		"password": "Passw0rd!",
	})

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body = assertNotOK(t, w)
	assert.Equal(t, "signup.invalid_email", body["msg"])

	// invalid password
	req, err = postFormReq("/api/auth/signup", map[string]string{
		"username": "tester002",
		"email":    "tester002@test.com",
		"password": "Passw123", // password strengh not enough
	})

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body = assertNotOK(t, w)
	// assert.Equal(t, "signup.invalid_password", body["msg"])

	// invalid password
	req, err = postFormReq("/api/auth/signup", map[string]string{
		"username": "tester002",
		"email":    "tester002@test.com",
		"password": "122222233333334444444", // password strengh not enough, again
	})

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body = assertNotOK(t, w)

	// duplicated username or email
	req, err = postFormReq("/api/auth/signup", map[string]string{
		"username": "tester002",
		"email":    "tester@test.com",
		"password": "Passw0rd!",
	})

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertNotOK(t, w)

	req, err = postFormReq("/api/auth/signup", map[string]string{
		"username": "tester009",
		"email":    "tester002@test.com",
		"password": "Passw0rd!",
	})

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertNotOK(t, w)
}

func TestLoginHandler(t *testing.T) {
	testSetup()

	defer func() {
		err := delUsrByName("tester001")
		if err != nil {
			t.Fatal(err)
		}
	}()

	// signup test user first
	req, err := postFormReq("/api/auth/signup", map[string]string{
		"username": "tester001",
		"email":    "tester001@test.com",
		"password": "Passw0rd!",
	})
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertOK(t, w)

	// login normally
	req, err = postFormReq("/api/auth/login", map[string]string{
		"id":       "tester001",
		"password": "Passw0rd!",
	})
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertOK(t, w)

	// login with email
	req, err = postFormReq("/api/auth/login", map[string]string{
		"id":       "tester001@test.com",
		"password": "Passw0rd!",
	})
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertOK(t, w)

	// check jwt_token in the cookies
	var token *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "jwt_token" {
			token = c
		}
	}
	assert.NotNil(t, token)

	// l.Infoln("jwt_token", token.Value, "jwt_key", viper.GetString("jwt_key"))
	// assert parse token
	_, err = jwt.ParseWithClaims(token.Value, &claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("jwt_key")), nil
	})

	assert.Nil(t, err)

	// login with bad username
	req, err = postFormReq("/api/auth/login", map[string]string{
		"id":       "tester001_@tt.com",
		"password": "Passw0rd!",
	})
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := assertNotOK(t, w)
	assert.Equal(t, "login.failed", body["msg"])

	// login with bad password
	req, err = postFormReq("/api/auth/login", map[string]string{
		"id":       "tester001@test.com",
		"password": "Passw0",
	})
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body = assertNotOK(t, w)
	assert.Equal(t, "login.failed", body["msg"])

	// login with not matched password
	req, err = postFormReq("/api/auth/login", map[string]string{
		"id":       "tester001@test.com",
		"password": "Passw0rddww!!",
	})
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body = assertNotOK(t, w)
	assert.Equal(t, "login.failed", body["msg"])
}

func TestJwtAuth(t *testing.T) {
	testSetup()

	defer func() {
		err := delUsrByName("tester")
		if err != nil {
			t.Fatal(err)
		}
	}()

	// signup test user first
	w := testLogin(t, "tester")

	// check jwt_token in the cookies
	token, _ := testJwtToken(t, w)

	// create a test router,  with jwt auth required
	r.GET("/api/auth/test", jwtAuth, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"username": c.GetString("username"),
			"email":    c.GetString("email"),
		})
	})

	req, _ := http.NewRequest("GET", "/api/auth/test", nil)
	req.AddCookie(token)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// ok, if we attached the jwt token
	body := assertOK(t, w)
	assert.Equal(t, "tester", body["username"])
	assert.Equal(t, "tester@test.com", body["email"])

	// not ok, if we attached the invalid jwt token
	token.Value = "invalid-token" // invalid jwt token
	req, _ = http.NewRequest("GET", "/api/auth/test", nil)
	req.AddCookie(token)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertNotOK(t, w)
}
