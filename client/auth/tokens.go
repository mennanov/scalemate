package auth

import (
	"github.com/gogo/protobuf/proto"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
)

// SaveTokens saves AuthTokens to a temporary file.
func SaveTokens(tokens *accounts_proto.AuthTokens) error {
	serializedTokens, err := proto.Marshal(tokens)
	if err != nil {
		return err
	}
	tokensPath := getTokensFilePath()
	logrus.Debugf("Saving auth tokens to %s", tokensPath)
	return ioutil.WriteFile(tokensPath, serializedTokens, 0600)
}

// LoadTokens loads AuthTokens from a temporary file.
func LoadTokens() (*accounts_proto.AuthTokens, error) {
	tokensPath := getTokensFilePath()
	logrus.Debugf("Loading auth tokens from %s", tokensPath)
	serializedTokens, err := ioutil.ReadFile(tokensPath)
	if err != nil {
		return nil, err
	}
	tokens := &accounts_proto.AuthTokens{}
	if err := proto.Unmarshal(serializedTokens, tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

// DeleteTokens deletes the token temporary file.
func DeleteTokens() error {
	tokensPath := getTokensFilePath()
	return os.Remove(tokensPath)
}

func getTokensFilePath() string {
	return path.Join(os.TempDir(), "scalemate_auth_tokens")
}
