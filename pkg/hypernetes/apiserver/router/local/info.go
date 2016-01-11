package local

import (
	"net/http"

	"k8s.io/kubernetes/pkg/version"

	"github.com/emicklei/go-restful"
)

func getVersion(req *restful.Request, resp *restful.Response) {
	writeRawJSON(http.StatusOK, version.Get(), resp.ResponseWriter)
}

func getInfo(req *restful.Request, resp *restful.Response) {
}

func getEvents(req *restful.Request, resp *restful.Response) {
}
