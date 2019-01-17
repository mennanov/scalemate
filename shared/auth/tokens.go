package auth

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/sirupsen/logrus"
)

// SaveTokens saves AuthTokens to a file.
func SaveTokens(tokens *accounts_proto.AuthTokens) error {
	serializedTokens, err := proto.Marshal(tokens)
	if err != nil {
		return err
	}
	tokensDir, err := getOrCreateTokensDir()
	if err != nil {
		return err
	}
	tokensPath := getTokensFilePath(tokensDir)
	logrus.Debugf("Saving auth tokens to %s", tokensPath)
	return ioutil.WriteFile(tokensPath, serializedTokens, 0600)
}

// LoadTokens loads AuthTokens from a file.
func LoadTokens() (*accounts_proto.AuthTokens, error) {
	tokensPath := getTokensFilePath(getTokensDir())
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

// DeleteTokens deletes the token file.
func DeleteTokens() error {
	tokensPath := getTokensFilePath(getTokensDir())
	return os.Remove(tokensPath)
}

func getTokensDir() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	return path.Join(cacheDir, "scalemate")
}

func getOrCreateTokensDir() (string, error) {
	tokensDir := getTokensDir()
	if _, err := os.Stat(tokensDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(tokensDir, 0700); err != nil {
				return "", errors.Wrapf(err, "failed to create a directory %s", tokensDir)
			}
		} else {
			return "", err
		}
	}
	return tokensDir, nil
}

func getTokensFilePath(tokensDir string) string {
	return path.Join(tokensDir, "auth_tokens")
}
