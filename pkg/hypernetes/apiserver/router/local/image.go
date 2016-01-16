package local

import (
	"net/http"

	"k8s.io/kubernetes/pkg/hypernetes/httputils"
	"k8s.io/kubernetes/pkg/hypernetes/storage"

	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/registry"
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

/*
	Query Parameters:
		term – term to search

	Status Codes:
		200 – no error
		500 – server error
*/
func (i *imageAction) getImagesSearch(req *restful.Request, resp *restful.Response) {
	query := registry.SearchResults{}
	httputils.WriteRawJSON(http.StatusOK, query.Results, resp.ResponseWriter)
}

/*
	Query Parameters:
		all – 1/True/true or 0/False/false, default false
		filters – a JSON encoded value of the filters (a map[string][]string) to process on the images list. Available filters:
			dangling=true
			label=key or label="key=value" of an image label
		filter - only return images with the specified name
*/
func (i *imageAction) getImagesJSON(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	if name != "" {
		imageJSON := types.ImageInspect{}
		httputils.WriteRawJSON(http.StatusOK, imageJSON, resp.ResponseWriter)
	} else {
		result := []types.Image{}
		httputils.WriteRawJSON(http.StatusOK, result, resp.ResponseWriter)
	}
}

/*
	Status Codes:
		200 – no error
		404 – no such image
		500 – server error
*/
func (i *imageAction) getImagesHistory(req *restful.Request, resp *restful.Response) {
	result := []types.ImageHistory{}
	httputils.WriteRawJSON(http.StatusOK, result, resp.ResponseWriter)
}

func (i *imageAction) getImagesByName(req *restful.Request, resp *restful.Response) {
}

func (i *imageAction) postCommit(req *restful.Request, resp *restful.Response) {
}

/*
	Query Parameters:
		fromImage – Name of the image to pull. The name may include a tag or digest. This parameter may only be used when pulling an image.
		fromSrc – Source to import. The value may be a URL from which the image can be retrieved or - to read the image from the request body. This parameter may only be used when importing an image.
		repo – Repository name given to an image when it is imported. The repo may include a tag. This parameter may only be used when importing an image.
		tag – Tag or digest.

	Request Headers:
		X-Registry-Auth – base64-encoded AuthConfig object

	Status Codes:
		200 – no error
		500 – server error
*/
func (i *imageAction) postImagesCreate(req *restful.Request, resp *restful.Response) {
}

/*
	Query Parameters:
		tag – The tag to associate with the image on the registry. This is optional.

	Request Headers:
		X-Registry-Auth – Include a base64-encoded AuthConfig. object.

	Status Codes:
		200 – no error
		404 – no such image
		500 – server error
*/
func (i *imageAction) postImagesPush(req *restful.Request, resp *restful.Response) {
}

func (i *imageAction) postImagesLoad(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

/*
	Query Parameters:
		repo – The repository to tag in
		force – 1/True/true or 0/False/false, default false
		tag - The new tag name

	Status Codes:
		201 – no error
		400 – bad parameter
		404 – no such image
		409 – conflict
		500 – server error
*/
func (i *imageAction) postImagesTag(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusCreated, nil, resp.ResponseWriter)
}

/*
	Query Parameters:
		force – 1/True/true or 0/False/false, default false
		noprune – 1/True/true or 0/False/false, default false

	Status Codes:
		200 – no error
		404 – no such image
		409 – conflict
		500 – server error
*/
func (i *imageAction) deleteImages(req *restful.Request, resp *restful.Response) {
	result := types.ImageDelete{}
	httputils.WriteRawJSON(http.StatusOK, result, resp.ResponseWriter)
}
