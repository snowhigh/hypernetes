package local

import (
	"net/http"

	"k8s.io/kubernetes/pkg/hypernetes/httputils"
	"k8s.io/kubernetes/pkg/version"

	"github.com/emicklei/go-restful"
)

func HandleMiscAction(verb, action string) restful.RouteFunction {
	switch verb {
	case "GET":
		switch action {
		case "_ping":
			return unsupportedAction
		case "events":
			return getEvents
		case "info":
			return getInfo
		case "version":
			return getVersion
		}
	case "POST":
		switch action {
		case "auth":
			return postAuth
		case "commit":
		case "build":
			return unsupportedAction
		}
		break
	}
	return unsupportedAction
}

func getVersion(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusOK, version.Get(), resp.ResponseWriter)
}

func getInfo(req *restful.Request, resp *restful.Response) {
}

func getEvents(req *restful.Request, resp *restful.Response) {
}

func unsupportedAction(req *restful.Request, resp *restful.Response) {
}
