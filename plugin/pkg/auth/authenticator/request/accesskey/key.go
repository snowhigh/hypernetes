package accesskey

import (
	"net/http"
	"strings"

	"k8s.io/kubernetes/pkg/auth/user"
	"k8s.io/kubernetes/pkg/hypernetes/auth"
	"k8s.io/kubernetes/pkg/hypernetes/httputils"
	"k8s.io/kubernetes/pkg/hypernetes/storage"

	"github.com/golang/glog"
)

// Accesskey authenticator
type AccesskeyAuthenticator struct {
	storage storage.Interface
}

// AuthenticateRequest
func (a *AccesskeyAuthenticator) AuthenticateRequest(req *http.Request) (user.Info, bool, error) {
	authorization := strings.TrimSpace(req.Header.Get("Authorization"))
	if authorization == "" {
		return nil, false, nil
	}
	parts := strings.Split(authorization, " ")
	if len(parts) < 2 || strings.ToLower(parts[0]) != "HSC" {
		return nil, false, nil
	}
	fields := strings.Split(parts[0], ":")
	if len(parts) != 2 {
		return nil, false, nil
	}
	accesskey, signatureRequest := fields[0], fields[1]
	var authReq auth.AuthItem
	err := a.storage.Get(auth.Database, auth.AuthTable, "accesskey", accesskey, &authReq)
	if err != nil {
		glog.Error(err)
		return nil, false, err
	}
	if signature, err := httputils.GetSign(&authReq, req); err != nil {
		glog.Error(err)
		return nil, false, err
	} else {
		if signature != signatureRequest {
			return nil, false, nil
		}
	}

	return &user.DefaultInfo{Name: authReq.UserID, Tenant: authReq.TenantID}, true, nil
}

// NewAccesskeyAuthenticator
func NewAccesskeyAuthenticator(storage storage.Interface) (*AccesskeyAuthenticator, error) {
	return &AccesskeyAuthenticator{
		storage: storage,
	}, nil
}
