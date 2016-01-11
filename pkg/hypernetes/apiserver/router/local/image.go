package local

import (
	"net/http"

	"k8s.io/kubernetes/pkg/hypernetes/httputils"

	"github.com/emicklei/go-restful"
)

func HandleImagesAction(verb, action string, hasParams bool) restful.RouteFunction {
	switch verb {
	case "GET":
		switch action {
		case "json":
			return getImagesJSON
		case "search":
			return getImagesSearch
		case "get":
			return getImagesGet
		case "history":
			return getImagesHistory
		}
	case "POST":
		switch action {
		case "create":
			return postImagesCreate
		case "load":
			return postImagesLoad
		case "push":
			return postImagesPush
		case "tag":
			return postImagesTag
		}
	case "DELETE":
		return deleteImages
	}
	return nil
}

func getImagesGet(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusOK, nil, resp.ResponseWriter)
}

func getImagesSearch(req *restful.Request, resp *restful.Response) {
}

func getImagesJSON(req *restful.Request, resp *restful.Response) {
}

func getImagesHistory(req *restful.Request, resp *restful.Response) {
}

func getImagesByName(req *restful.Request, resp *restful.Response) {
}

func postCommit(req *restful.Request, resp *restful.Response) {
}

func postImagesCreate(req *restful.Request, resp *restful.Response) {
}

func postImagesPush(req *restful.Request, resp *restful.Response) {
}

func postImagesLoad(req *restful.Request, resp *restful.Response) {
}

func postImagesTag(req *restful.Request, resp *restful.Response) {
}

func deleteImages(req *restful.Request, resp *restful.Response) {
}
