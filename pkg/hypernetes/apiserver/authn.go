package apiserver

import (
	"k8s.io/kubernetes/pkg/auth/authenticator"
	"k8s.io/kubernetes/pkg/hypernetes/storage"
	"k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/accesskey"
)

type AuthenticatorConfig struct {
	Storage     storage.Interface
	KeystoneURL string
}

// NewAuthenticator returns an authenticator.Request or an error
func NewAuthenticator(config AuthenticatorConfig) (authenticator.Request, error) {

	accesskeyAuth, err := accesskey.NewAccesskeyAuthenticator(config.Storage)
	if err != nil {
		return nil, err
	}
	return accesskeyAuth, nil
}
