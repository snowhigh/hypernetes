package local

import "github.com/emicklei/go-restful"

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
}

func getContainersStats(req *restful.Request, resp *restful.Response) {
}

func getContainersLogs(req *restful.Request, resp *restful.Response) {
}

func getContainersExport(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

func getContainersChanges(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

func getContainersTop(req *restful.Request, resp *restful.Response) {
}

func getContainersArchive(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

func postContainersStart(req *restful.Request, resp *restful.Response) {
}

func postContainersStop(req *restful.Request, resp *restful.Response) {
}

func postContainersKill(req *restful.Request, resp *restful.Response) {
}

func postContainersRestart(req *restful.Request, resp *restful.Response) {
}

func postContainersPause(req *restful.Request, resp *restful.Response) {
}

func postContainersUnpause(req *restful.Request, resp *restful.Response) {
}

func postContainersWait(req *restful.Request, resp *restful.Response) {
}

func postContainerRename(req *restful.Request, resp *restful.Response) {
}

func postContainersCreate(req *restful.Request, resp *restful.Response) {
}

func postContainersAttach(req *restful.Request, resp *restful.Response) {
}

func postContainersResize(req *restful.Request, resp *restful.Response) {
}

func postContainersCopy(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

func postContainersExec(req *restful.Request, resp *restful.Response) {
}

func putContainersArchive(req *restful.Request, resp *restful.Response) {
	unsupportedAction(req, resp)
}

func deleteContainers(req *restful.Request, resp *restful.Response) {
}
