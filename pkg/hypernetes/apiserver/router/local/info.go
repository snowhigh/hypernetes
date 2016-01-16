package local

import (
	"net/http"
	"runtime"

	"k8s.io/kubernetes/pkg/hypernetes/httputils"

	"github.com/docker/engine-api/types"
	"github.com/emicklei/go-restful"
)

func HandleMiscAction(verb, action string) restful.RouteFunction {
	switch verb {
	case "GET":
		switch action {
		case "_ping":
			return unsupportedAction
		case "events":
			return unsupportedAction
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
	version := types.Version{
		Version:    "1.91",
		APIVersion: "1.21",
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}
	httputils.WriteRawJSON(http.StatusOK, version, resp.ResponseWriter)
}

func getInfo(req *restful.Request, resp *restful.Response) {
	info := types.Info{}
	httputils.WriteRawJSON(http.StatusOK, info, resp.ResponseWriter)
}

func unsupportedAction(req *restful.Request, resp *restful.Response) {
	httputils.NotSupport(resp, req.Request)
}
