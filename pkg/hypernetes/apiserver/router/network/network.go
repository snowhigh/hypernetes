package network

import (
	"net/http"

	"k8s.io/kubernetes/pkg/hypernetes/httputils"

	"github.com/docker/engine-api/types"
	"github.com/emicklei/go-restful"
)

func HandleNetworkAction(verb, action string) restful.RouteFunction {
	switch verb {
	case "GET":
		return getNetwork
	case "POST":
		switch action {
		case "create":
			return postNetworkCreate
		case "connect":
			return postNetworkConnect
		case "disconnect":
			return postNetworkDisconnect
		}
	case "DELETE":
		return deleteNetwork
	}
	return unsupportedAction
}

/*
	Query Parameters:
		filters - JSON encoded value of the filters (a map[string][]string) to process on the networks list. Available filters: name=[network-names] , id=[network-ids]

	Status Codes:
		200 - no error
		500 - server error
*/
func getNetworksList(req *restful.Request, resp *restful.Response) {
	list := []types.NetworkResource{}
	httputils.WriteRawJSON(http.StatusOK, list, resp.ResponseWriter)
}

/*
	Status Codes:
		200 - no error
		404 - network not found
*/
func getNetwork(req *restful.Request, resp *restful.Response) {
	networkID := req.PathParameter("id")
	if networkID == "" {
		getNetworksList(req, resp)
		return
	}
	result := types.NetworkResource{}
	httputils.WriteRawJSON(http.StatusOK, result, resp.ResponseWriter)
}

/*
	Status Codes:
		201 - no error
		404 - driver not found
		500 - server error
*/
func postNetworkCreate(req *restful.Request, resp *restful.Response) {
	networkCreateResp := types.NetworkCreateResponse{}
	httputils.WriteRawJSON(http.StatusCreated, networkCreateResp, resp.ResponseWriter)
}

/*
	Status Codes:
		200 - no error
		404 - network or container is not found
*/
func postNetworkConnect(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusOK, nil, resp.ResponseWriter)
}

/*
	Status Codes:
		200 - no error
		404 - network or container is not found
*/
func postNetworkDisconnect(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusOK, nil, resp.ResponseWriter)
}

/*
	Status Codes
		204 - no error
		404 - no such network
		500 - server error
*/
func deleteNetwork(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}

func unsupportedAction(req *restful.Request, resp *restful.Response) {
	httputils.NotSupport(resp, req.Request)
}
