package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"root/constant"
)

func VerifyGoogleToken(accessToken, userEmail, userName string) (constant.GoogleTokenInfo, error) {
	resp, err := http.Get(fmt.Sprintf("https://www.googleapis.com/oauth2/v3/userinfo?access_token=%s", accessToken))
	if err != nil {
		return constant.GoogleTokenInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return constant.GoogleTokenInfo{}, errors.New("invalid token")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return constant.GoogleTokenInfo{}, err
	}

	var tokenInfo constant.GoogleTokenInfo
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return constant.GoogleTokenInfo{}, err
	}

	if tokenInfo.Email != userEmail || tokenInfo.Name != userName {
		return constant.GoogleTokenInfo{}, errors.New("token info does not match user credentials")
	}

	return tokenInfo, nil
}
