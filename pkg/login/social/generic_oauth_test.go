package social

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/grafana/grafana/pkg/infra/log"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/oauth2"
	"testing"
)

func TestSearchJSONForEmail(t *testing.T) {
	Convey("Given a generic OAuth provider", t, func() {
		provider := SocialGenericOAuth{
			SocialBase: &SocialBase{
				log: log.New("generic_oauth_test"),
			},
		}

		tests := []struct {
			Name                 string
			UserInfoJSONResponse []byte
			EmailAttributePath   string
			ExpectedResult       string
		}{
			{
				Name:                 "Given an invalid user info JSON response",
				UserInfoJSONResponse: []byte("{"),
				EmailAttributePath:   "attributes.email",
				ExpectedResult:       "",
			},
			{
				Name:                 "Given an empty user info JSON response and empty JMES path",
				UserInfoJSONResponse: []byte{},
				EmailAttributePath:   "",
				ExpectedResult:       "",
			},
			{
				Name:                 "Given an empty user info JSON response and valid JMES path",
				UserInfoJSONResponse: []byte{},
				EmailAttributePath:   "attributes.email",
				ExpectedResult:       "",
			},
			{
				Name: "Given a simple user info JSON response and valid JMES path",
				UserInfoJSONResponse: []byte(`{
	"attributes": {
		"email": "grafana@localhost"
	}
}`),
				EmailAttributePath: "attributes.email",
				ExpectedResult:     "grafana@localhost",
			},
			{
				Name: "Given a user info JSON response with e-mails array and valid JMES path",
				UserInfoJSONResponse: []byte(`{
	"attributes": {
		"emails": ["grafana@localhost", "admin@localhost"]
	}
}`),
				EmailAttributePath: "attributes.emails[0]",
				ExpectedResult:     "grafana@localhost",
			},
			{
				Name: "Given a nested user info JSON response and valid JMES path",
				UserInfoJSONResponse: []byte(`{
	"identities": [
		{
			"userId": "grafana@localhost"
		},
		{
			"userId": "admin@localhost"
		}
	]
}`),
				EmailAttributePath: "identities[0].userId",
				ExpectedResult:     "grafana@localhost",
			},
		}

		for _, test := range tests {
			provider.emailAttributePath = test.EmailAttributePath
			Convey(test.Name, func() {
				actualResult := provider.searchJSONForAttr(test.EmailAttributePath, test.UserInfoJSONResponse)
				So(actualResult, ShouldEqual, test.ExpectedResult)
			})
		}
	})
}

func TestSearchJSONForRole(t *testing.T) {
	Convey("Given a generic OAuth provider", t, func() {
		provider := SocialGenericOAuth{
			SocialBase: &SocialBase{
				log: log.New("generic_oauth_test"),
			},
		}

		tests := []struct {
			Name                 string
			UserInfoJSONResponse []byte
			RoleAttributePath    string
			ExpectedResult       string
		}{
			{
				Name:                 "Given an invalid user info JSON response",
				UserInfoJSONResponse: []byte("{"),
				RoleAttributePath:    "attributes.role",
				ExpectedResult:       "",
			},
			{
				Name:                 "Given an empty user info JSON response and empty JMES path",
				UserInfoJSONResponse: []byte{},
				RoleAttributePath:    "",
				ExpectedResult:       "",
			},
			{
				Name:                 "Given an empty user info JSON response and valid JMES path",
				UserInfoJSONResponse: []byte{},
				RoleAttributePath:    "attributes.role",
				ExpectedResult:       "",
			},
			{
				Name: "Given a simple user info JSON response and valid JMES path",
				UserInfoJSONResponse: []byte(`{
	"attributes": {
		"role": "admin"
	}
}`),
				RoleAttributePath: "attributes.role",
				ExpectedResult:    "admin",
			},
		}

		for _, test := range tests {
			provider.roleAttributePath = test.RoleAttributePath
			Convey(test.Name, func() {
				actualResult := provider.searchJSONForAttr(test.RoleAttributePath, test.UserInfoJSONResponse)
				So(actualResult, ShouldEqual, test.ExpectedResult)
			})
		}
	})
}

func TestUserInfoSearchesForEmailAndRole(t *testing.T) {
	Convey("Given a generic OAuth provider", t, func() {
		provider := SocialGenericOAuth{
			SocialBase: &SocialBase{
				log: log.New("generic_oauth_test"),
			},
			emailAttributePath: "email",
		}

		tests := []struct {
			Name              string
			OAuth2Token       oauth2.Token
			APIURLReponse     interface{}
			OAuth2Extra       interface{}
			RoleAttributePath string
			ExpectedEmail     string
			ExpectedRole      string
		}{
			{
				Name: "Given a valid id_token, a valid role path, no api response --> use id_token",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				OAuth2Extra: map[string]interface{}{
					// { "role": "Admin", "email": "john.doe@example.com" }
					"id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoiQWRtaW4iLCJlbWFpbCI6ImpvaG4uZG9lQGV4YW1wbGUuY29tIn0.9PtHcCaXxZa2HDlASyKIaFGfOKlw2ILQo32xlvhvhRg",
				},
				RoleAttributePath: "role",
				ExpectedEmail:     "john.doe@example.com",
				ExpectedRole:      "Admin",
			},
			{
				Name: "Given a valid id_token, no role path, no api response --> use id_token",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				OAuth2Extra: map[string]interface{}{
					// { "email": "john.doe@example.com" }
					"id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImpvaG4uZG9lQGV4YW1wbGUuY29tIn0.k5GwPcZvGe2BE_jgwN0ntz0nz4KlYhEd0hRRLApkTJ4",
				},
				RoleAttributePath: "",
				ExpectedEmail:     "john.doe@example.com",
				ExpectedRole:      "",
			},
			{
				Name: "Given a valid id_token, an invalid role path, no api response --> use id_token",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				OAuth2Extra: map[string]interface{}{
					// { "role": "Admin", "email": "john.doe@example.com" }
					"id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoiQWRtaW4iLCJlbWFpbCI6ImpvaG4uZG9lQGV4YW1wbGUuY29tIn0.9PtHcCaXxZa2HDlASyKIaFGfOKlw2ILQo32xlvhvhRg",
				},
				RoleAttributePath: "invalid_path",
				ExpectedEmail:     "john.doe@example.com",
				ExpectedRole:      "",
			},
			{
				Name: "Given no id_token, a valid role path, a valid api response --> use api response",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				APIURLReponse: map[string]interface{}{
					"role": "Admin",
					"email": "john.doe@example.com",
				},
				RoleAttributePath: "role",
				ExpectedEmail:     "john.doe@example.com",
				ExpectedRole:      "Admin",
			},
			{
				Name: "Given no id_token, no role path, a valid api response --> use api response",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				APIURLReponse: map[string]interface{}{
					"email": "john.doe@example.com",
				},
				RoleAttributePath: "",
				ExpectedEmail:     "john.doe@example.com",
				ExpectedRole:      "",
			},
			{
				Name: "Given no id_token, a role path, a valid api response without a role --> use api response",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				APIURLReponse: map[string]interface{}{
					"email": "john.doe@example.com",
				},
				RoleAttributePath: "role",
				ExpectedEmail:     "john.doe@example.com",
				ExpectedRole:      "",
			},
			{
				Name: "Given no id_token, a valid role path, no api response --> no data",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				RoleAttributePath: "role",
				ExpectedEmail:     "",
				ExpectedRole:      "",
			},
			{
				Name: "Given a valid id_token, a valid role path, a valid api response -> prefer id_token",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				OAuth2Extra: map[string]interface{}{
					// { "role": "Admin", "email": "john.doe@example.com" }
					"id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoiQWRtaW4iLCJlbWFpbCI6ImpvaG4uZG9lQGV4YW1wbGUuY29tIn0.9PtHcCaXxZa2HDlASyKIaFGfOKlw2ILQo32xlvhvhRg",
				},
				APIURLReponse: map[string]interface{}{
					"role":  "FromResponse",
					"email": "from_response@example.com",
				},
				RoleAttributePath: "role",
				ExpectedEmail:     "john.doe@example.com",
				ExpectedRole:      "Admin",
			},
			{
				Name: "Given a valid id_token, an invalid role path, a valid api response -> prefer id_token",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				OAuth2Extra: map[string]interface{}{
					// { "role": "Admin", "email": "john.doe@example.com" }
					"id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoiQWRtaW4iLCJlbWFpbCI6ImpvaG4uZG9lQGV4YW1wbGUuY29tIn0.9PtHcCaXxZa2HDlASyKIaFGfOKlw2ILQo32xlvhvhRg",
				},
				APIURLReponse: map[string]interface{}{
					"role":  "FromResponse",
					"email": "from_response@example.com",
				},
				RoleAttributePath: "invalid_path",
				ExpectedEmail:     "john.doe@example.com",
				ExpectedRole:      "",
			},
			{
				Name: "Given a valid id_token with no email, a valid role path, a valid api response with no role --> merge",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				OAuth2Extra: map[string]interface{}{
					// { "role": "Admin" }
					"id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoiQWRtaW4ifQ.k5GwPcZvGe2BE_jgwN0ntz0nz4KlYhEd0hRRLApkTJ4",
				},
				APIURLReponse: map[string]interface{}{
					"email": "from_response@example.com",
				},
				RoleAttributePath: "role",
				ExpectedEmail:     "from_response@example.com",
				ExpectedRole:      "Admin",
			},
			{
				Name: "Given a valid id_token with no role, a valid role path, a valid api response with no email --> merge",
				OAuth2Token: oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Now(),
				},
				OAuth2Extra: map[string]interface{}{
					// { "email": "john.doe@example.com" }
					"id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImpvaG4uZG9lQGV4YW1wbGUuY29tIn0.k5GwPcZvGe2BE_jgwN0ntz0nz4KlYhEd0hRRLApkTJ4",
				},
				APIURLReponse: map[string]interface{}{
					"role": "FromResponse",
				},
				RoleAttributePath: "role",
				ExpectedEmail:     "john.doe@example.com",
				ExpectedRole:      "FromResponse",
			},
		}

		for _, test := range tests {
			provider.roleAttributePath = test.RoleAttributePath
			Convey(test.Name, func() {
				response, _ := json.Marshal(test.APIURLReponse)
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, string(response))
				}))
				provider.apiUrl = ts.URL
				token := test.OAuth2Token.WithExtra(test.OAuth2Extra)
				actualResult, _ := provider.UserInfo(ts.Client(), token)
				So(actualResult.Email, ShouldEqual, test.ExpectedEmail)
				So(actualResult.Role, ShouldEqual, test.ExpectedRole)
			})
		}
	})
}
