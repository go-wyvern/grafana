package social

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-wyvern/grafana/pkg/models"

	"golang.org/x/oauth2"
)

type SocialGoogle struct {
	*oauth2.Config
	allowedDomains []string
	hostedDomain   string
	apiUrl         string
	allowSignup    bool
}

func (s *SocialGoogle) Type() int {
	return int(models.GOOGLE)
}

func (s *SocialGoogle) IsEmailAllowed(email string) bool {
	return isEmailAllowed(email, s.allowedDomains)
}

func (s *SocialGoogle) IsSignupAllowed() bool {
	return s.allowSignup
}

func (s *SocialGoogle) UserInfo(client *http.Client) (*BasicUserInfo, error) {
	var data struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	response, err := HttpGet(client, s.apiUrl)
	if err != nil {
		return nil, fmt.Errorf("Error getting user info: %s", err)
	}

	err = json.Unmarshal(response.Body, &data)
	if err != nil {
		return nil, fmt.Errorf("Error getting user info: %s", err)
	}

	return &BasicUserInfo{
		Name:  data.Name,
		Email: data.Email,
		Login: data.Email,
	}, nil
}
