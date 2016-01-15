package local

import (
	"net/http"

	"k8s.io/kubernetes/pkg/hypernetes/auth"
	"k8s.io/kubernetes/pkg/hypernetes/httputils"
	"k8s.io/kubernetes/pkg/hypernetes/storage"

	"github.com/emicklei/go-restful"
)

type imageAction struct {
	stor storage.Interface
}

func HandleImagesAction(verb, action string, hasParams bool, stor storage.Interface) restful.RouteFunction {
	i := &imageAction{
		stor: stor,
	}
	switch verb {
	case "GET":
		switch action {
		case "json":
			return i.getImagesJSON
		case "search":
			return i.getImagesSearch
		case "get":
			return i.getImagesGet
		case "history":
			return i.getImagesHistory
		}
	case "POST":
		switch action {
		case "create":
			return i.postImagesCreate
		case "load":
			return i.postImagesLoad
		case "push":
			return i.postImagesPush
		case "tag":
			return i.postImagesTag
		}
	case "DELETE":
		return i.deleteImages
	}
	return nil
}

func (i *imageAction) getImagesGet(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

func (i *imageAction) getImagesSearch(req *restful.Request, resp *restful.Response) {
}

func (i *imageAction) getImagesJSON(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	httputils.WriteRawJSON(http.StatusOK, name+"\n", resp.ResponseWriter)
	i.stor.Delete("hypernetes", "auth", "accesskey", "1")
	if err := i.stor.Create("hypernetes", "auth", &auth.AuthItem{AccessKey: "1", SecretKey: "haha"}); err != nil {
		httputils.WriteRawJSON(http.StatusInternalServerError, err, resp.ResponseWriter)
		return
	}
	var result auth.AuthItem
	if err := i.stor.Get("hypernetes", "auth", "accesskey", "1", &result); err != nil {
		httputils.WriteRawJSON(http.StatusInternalServerError, err, resp.ResponseWriter)
		return
	} else {
		httputils.WriteRawJSON(http.StatusOK, result, resp.ResponseWriter)
	}
}

func (i *imageAction) getImagesHistory(req *restful.Request, resp *restful.Response) {
}

func (i *imageAction) getImagesByName(req *restful.Request, resp *restful.Response) {
}

func (i *imageAction) postCommit(req *restful.Request, resp *restful.Response) {
}

func (i *imageAction) postImagesCreate(req *restful.Request, resp *restful.Response) {
}

func (i *imageAction) postImagesPush(req *restful.Request, resp *restful.Response) {
}

func (i *imageAction) postImagesLoad(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusOK, nil, resp.ResponseWriter)
}

func (i *imageAction) postImagesTag(req *restful.Request, resp *restful.Response) {
}

func (i *imageAction) deleteImages(req *restful.Request, resp *restful.Response) {
}
