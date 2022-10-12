package dream

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var setupOnlyOnce sync.Once

func testSetup() {
	setupOnlyOnce.Do(func() {
		router := gin.Default()
		Config()
		Setup(router)
	})
}

func postFormReq(addr string, data map[string]string) (req *http.Request, err error) {
	formData := url.Values{}
	for key, value := range data {
		formData[key] = []string{value}
	}

	req, err = http.NewRequest("POST", addr, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return
}

func postJsonReq(addr string, data interface{}) (req *http.Request, err error) {
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err = http.NewRequest("POST", addr, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	return
}

func assertOK(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	assert.Equal(t, w.Result().StatusCode, http.StatusOK)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.Nil(t, err)

	assert.Equal(t, true, body["ok"].(bool))

	return body
}

func assertNotOK(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	assert.NotEqual(t, w.Result().StatusCode, http.StatusOK)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.Nil(t, err)

	assert.Equal(t, false, body["ok"].(bool))

	return body
}

func TestPingHandle(t *testing.T) {
	testSetup()

	req, _ := http.NewRequest("GET", "/api/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertOK(t, w)

	req, _ = http.NewRequest("GET", "/api/pong", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assertOK(t, w)
}
