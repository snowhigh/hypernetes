package local

import (
	"net/http"

	"k8s.io/kubernetes/pkg/hypernetes/httputils"

	"github.com/emicklei/go-restful"
)

/*
	Status Codes:
		200 – no error
		204 – no error
		500 – server error
*/
func postAuth(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusOK, nil, resp.ResponseWriter)
}
