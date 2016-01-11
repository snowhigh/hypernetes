package apiserver

import (
	"fmt"

	"k8s.io/kubernetes/pkg/auth/authenticator"
	"k8s.io/kubernetes/pkg/storage"
	"k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/basicauth"
	"k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/keystone"
)

type AuthenticatorConfig struct {
	Storage     storage.Interface
	KeystoneURL string
}

// NewAuthenticator returns an authenticator.Request or an error
func NewAuthenticator(config AuthenticatorConfig) (authenticator.Request, error) {

	if len(config.KeystoneURL) > 0 {
		keystoneAuth, err := newAuthenticatorFromKeystoneURL(config.KeystoneURL)
		if err != nil {
			return nil, err
		}
		return keystoneAuth, nil
	}
	return nil, fmt.Errorf("error without keystone URL")
}

// newAuthenticatorFromTokenFile returns an authenticator.Request or an error
func newAuthenticatorFromKeystoneURL(keystoneConfigFile string) (authenticator.Request, error) {
	keystoneAuthenticator, err := keystone.NewKeystoneAuthenticator(keystoneConfigFile)
	if err != nil {
		return nil, err
	}

	return basicauth.New(keystoneAuthenticator), nil
}
