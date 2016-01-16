package local

import (
	"net/http"

	"k8s.io/kubernetes/pkg/hypernetes/httputils"

	"github.com/docker/engine-api/types"
	"github.com/emicklei/go-restful"
)

func HandleContainersAction(verb, action string, hasParams bool) restful.RouteFunction {
	switch verb {
	case "GET":
		switch action {
		case "json":
			return getContainersJSON
		case "stats":
			return getContainersStats
		case "logs":
			return getContainersLogs
		case "export":
			return getContainersExport
		case "changes":
			return getContainersChanges
		case "top":
			return getContainersTop
		case "archive":
			return getContainersArchive
		}
		break
	case "POST":
		switch action {
		case "start":
			return postContainersStart
		case "stop":
			return postContainersStop
		case "kill":
			return postContainersKill
		case "restart":
			return postContainersRestart
		case "pause":
			return postContainersPause
		case "unpause":
			return postContainersUnpause
		case "wait":
			return postContainersWait
		case "rename":
			return postContainerRename
		case "create":
			return postContainersCreate
		case "attach":
			return postContainersAttach
		case "resize":
			return postContainersResize
		case "copy":
			return postContainersCopy
		case "exec":
			return postContainersExec
		}
		break
	case "PUT":
		return putContainersArchive
	case "DELETE":
		return deleteContainers
	}
	return nil
}

func getContainersJSON(req *restful.Request, resp *restful.Response) {
	containerJSON := types.ContainerJSON{}
	httputils.WriteRawJSON(http.StatusOK, containerJSON, resp.ResponseWriter)
}

/*
	Query Parameters:
		stream – 1/True/true or 0/False/false, pull stats once then disconnect. Default true.

	Status Codes:
		200 – no error
		404 – no such container
		500 – server error
*/
func getContainersStats(req *restful.Request, resp *restful.Response) {
	stats := types.StatsJSON{}
	httputils.WriteRawJSON(http.StatusOK, stats, resp.ResponseWriter)
}

/*
	Status Codes:
		101 – no error, hints proxy about hijacking
		200 – no error, no upgrade header found
		404 – no such container
		500 – server error
*/
func getContainersLogs(req *restful.Request, resp *restful.Response) {
}

func getContainersExport(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

func getContainersChanges(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

/*
	Query Parameters:
		ps_args – ps arguments to use (e.g., aux)

	Status Codes:
		200 – no error
		404 – no such container
		500 – server error
*/
func getContainersTop(req *restful.Request, resp *restful.Response) {
	topStats := types.ContainerProcessList{}
	httputils.WriteRawJSON(http.StatusOK, topStats, resp.ResponseWriter)
}

func getContainersArchive(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

func postContainersStart(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}

/*
	Query Parameters:
		t – number of seconds to wait before killing the container

	Status Codes:
		204 – no error
		304 – container already stopped
		404 – no such container
		500 – server error
*/
func postContainersStop(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}

/*
	Query Parameters
		signal - Signal to send to the container: integer or string like SIGINT. When not set, SIGKILL is assumed and the call waits for the container to exit.

	Status Codes:
		204 – no error
		404 – no such container
		500 – server error
*/
func postContainersKill(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}

/*
	Query Parameters:
		t – number of seconds to wait before killing the container

	Status Codes:
		204 – no error
		404 – no such container
		500 – server error
*/
func postContainersRestart(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}

/*
	Status Codes:
		204 – no error
		404 – no such container
		500 – server error
*/
func postContainersPause(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}

func postContainersUnpause(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}

func postContainersWait(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

/*
	Query Parameters:
		name – new name for the container

	Status Codes:
		204 – no error
		404 – no such container
		409 - conflict name already assigned
		500 – server error
*/
func postContainerRename(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}

/*
	Query Parameters:
		name – Assign the specified name to the container. Must match /?[a-zA-Z0-9_-]+.

	Status Codes:
		201 – no error
		404 – no such container
		406 – impossible to attach (container not running)
		500 – server error
*/
func postContainersCreate(req *restful.Request, resp *restful.Response) {
	response := types.ContainerCreateResponse{
		Warnings: []string{},
	}
	httputils.WriteRawJSON(http.StatusOK, response, resp.ResponseWriter)
}

func postContainersAttach(req *restful.Request, resp *restful.Response) {
}

/*
	Query Parameters:
		h – height of tty session
		w – width

	Status Codes:
		200 – no error
		404 – No such container
		500 – Cannot resize container
*/
func postContainersResize(req *restful.Request, resp *restful.Response) {
}

func postContainersCopy(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

func postContainersExec(req *restful.Request, resp *restful.Response) {
	response := types.ContainerExecCreateResponse{}
	httputils.WriteRawJSON(http.StatusOK, response, resp.ResponseWriter)
}

func putContainersArchive(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

/*
	Query Parameters:
		v – 1/True/true or 0/False/false, Remove the volumes associated to the container. Default false.
		force - 1/True/true or 0/False/false, Kill then remove the container. Default false.

	Status Codes:
		204 – no error
		400 – bad parameter
		404 – no such container
		500 – server error
*/
func deleteContainers(req *restful.Request, resp *restful.Response) {
	httputils.WriteRawJSON(http.StatusNoContent, nil, resp.ResponseWriter)
}
