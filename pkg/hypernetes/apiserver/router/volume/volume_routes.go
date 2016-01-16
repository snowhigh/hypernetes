package volume

import (
	"net/http"

	"k8s.io/kubernetes/pkg/hypernetes/httputils"

	"github.com/docker/engine-api/types"
	"github.com/emicklei/go-restful"
)

func HandleVolumeAction(verb, action string) restful.RouteFunction {
	switch verb {
	case "GET":
		return getVolume
	case "POST":
		return postVolumesCreate
	case "DELETE":
		return deleteVolumes
	}
	return unsupportedAction
}

/*
	Query Parameters:
		filters - JSON encoded value of the filters (a map[string][]string) to process on the volumes list. There is one available filter: dangling=true

	Status Codes:
		200 - no error
		500 - server error
*/
func getVolumesList(req *restful.Request, resp *restful.Response) {
	list := types.VolumesListResponse{}
	httputils.WriteRawJSON(http.StatusOK, list, resp.ResponseWriter)
}

/*
	Status Codes:
		200 - no error
		404 - no such volume
		500 - server error
*/
func getVolume(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	if name == "" {
		getVolumesList(req, resp)
		return
	}
	volume := types.Volume{}
	httputils.WriteRawJSON(http.StatusOK, volume, resp.ResponseWriter)
}

/*
	Status Codes:
		201 - no error
		500 - server error
*/
func postVolumesCreate(req *restful.Request, resp *restful.Response) {
	volume := types.Volume{}
	httputils.WriteRawJSON(http.StatusCreated, volume, resp.ResponseWriter)
}

/*
	Status Codes
		204 - no error
		404 - no such volume or volume driver
		409 - volume is in use and cannot be removed
		500 - server error
*/
func deleteVolumes(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}

func unsupportedAction(req *restful.Request, resp *restful.Response) {
	httputils.NotSupport(resp, req.Request)
}
