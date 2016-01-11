package apiserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/hypernetes/apiserver/router/local"

	"github.com/emicklei/go-restful"
)

type APIInstaller struct {
	group             *APIGroupVersion
	info              *RequestInfoResolver
	prefix            string // Path prefix where API resources are to be registered.
	minRequestTimeout time.Duration
}

// Struct capturing information about an action ("GET", "POST", "DELETE", etc).
type action struct {
	Verb   string               // Verb identifying the action ("GET", "POST", "DELETE", etc).
	Path   string               // The path of the action
	Params []*restful.Parameter // List of parameters associated with the action.
}

// Installs handlers for API resources.
func (a *APIInstaller) Install(ws *restful.WebService) (errors []error) {
	errors = make([]error, 0)

	err := a.registerImageHandlers(ws)
	if err != nil {
		errors = append(errors, err)
	}
	err = a.registerContainerHandlers(ws)
	if err != nil {
		errors = append(errors, err)
	}
	err = a.registerMiscHandlers(ws)
	if err != nil {
		errors = append(errors, err)
	}

	return errors
}

// NewWebService creates a new restful webservice with the api installer's prefix and version.
func (a *APIInstaller) NewWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path(a.prefix)
	// a.prefix contains "prefix/group/version"
	ws.Doc("API at " + a.prefix)
	// TODO: change to restful.MIME_JSON when we set content type in client
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)
	ws.ApiVersion(a.group.GroupVersion.String())

	return ws
}

/*
	// GET
		"/_ping"
		"/events"
		"/info"
		"/version"
	// POST
		"/auth"
		"/commit"
		"/build"
*/
func (a *APIInstaller) registerMiscHandlers(ws *restful.WebService) error {
	actions := []action{}

	actions = append(actions, action{"GET", "/_ping", nil})
	actions = append(actions, action{"GET", "/events", nil})
	actions = append(actions, action{"GET", "/info", nil})
	actions = append(actions, action{"GET", "/version", nil})

	actions = append(actions, action{"POST", "/auth", nil})
	actions = append(actions, action{"POST", "/commit", nil})
	actions = append(actions, action{"POST", "/build", nil})

	for _, action := range actions {
		m := monitorFilter(action.Verb, "misc")
		switch action.Verb {
		case "GET":
			doc := "read the misc resources"
			route := ws.GET(action.Path).To(local.HandleMiscAction(action.Verb, action.Path[1:])).
				Filter(m).
				Doc(doc).
				Operation("get"+action.Path[1:]).
				Consumes(restful.MIME_XML, restful.MIME_JSON).
				Produces(restful.MIME_XML, restful.MIME_JSON)
			ws.Route(route)
			break
		case "POST":
			doc := "update the misc resources"
			route := ws.POST(action.Path).To(local.HandleMiscAction(action.Verb, action.Path[1:])).
				Filter(m).
				Doc(doc).
				Operation("post"+action.Path[1:]).
				Consumes(restful.MIME_XML, restful.MIME_JSON).
				Produces(restful.MIME_XML, restful.MIME_JSON)
			ws.Route(route)
			break
		default:
			return fmt.Errorf("unsupported action")
		}
	}
	return nil
}

/*
	// GET
		"/images/json"
		"/images/search"
		"/images/get"
		"/images/{name:.*}/get"
		"/images/{name:.*}/history"
		"/images/{name:.*}/json"
	// POST
		"/images/create"
		"/images/load"
		"/images/{name:.*}/push"
		"/images/{name:.*}/tag"
	// DELETE
		"/images/{name:.*}"
*/
func (a *APIInstaller) registerImageHandlers(ws *restful.WebService) error {

	nameParam := ws.PathParameter("name", "name of the image").DataType("string")
	params := []*restful.Parameter{nameParam}
	actions := []action{}

	actions = append(actions, action{"GET", "/images/json", nil})
	actions = append(actions, action{"GET", "/images/search", nil})
	actions = append(actions, action{"GET", "/images/get", nil})
	actions = append(actions, action{"GET", "/images/{name}/get", params})
	actions = append(actions, action{"GET", "/images/{name}/history", params})
	actions = append(actions, action{"GET", "/images/{name}/json", params})

	actions = append(actions, action{"POST", "/images/create", nil})
	actions = append(actions, action{"POST", "/images/load", nil})
	actions = append(actions, action{"POST", "/images/{name}/push", params})
	actions = append(actions, action{"POST", "/images/{name}/tag", params})

	actions = append(actions, action{"DELETE", "/images/{name}", params})

	for _, action := range actions {
		m := monitorFilter(action.Verb, "images")
		hasParams := false
		if action.Params != nil {
			hasParams = true
		}
		fields := strings.Split(action.Path, "/")
		subAction := fields[len(fields)-1]
		switch action.Verb {
		case "GET":
			doc := "read the specified images"
			route := ws.GET(action.Path).To(local.HandleImagesAction(action.Verb, subAction, hasParams)).
				Filter(m).
				Doc(doc).
				Operation("getimages"+subAction).
				Consumes(restful.MIME_XML, restful.MIME_JSON).
				Produces(restful.MIME_XML, restful.MIME_JSON)
			addParams(route, action.Params)
			ws.Route(route)
			break
		case "POST":
			doc := "update the specified images"
			route := ws.POST(action.Path).To(local.HandleImagesAction(action.Verb, subAction, hasParams)).
				Filter(m).
				Doc(doc).
				Operation("postimages"+subAction).
				Consumes(restful.MIME_XML, restful.MIME_JSON).
				Produces(restful.MIME_XML, restful.MIME_JSON)
			addParams(route, action.Params)
			ws.Route(route)
			break
		case "DELETE":
			doc := "delete the specified image"
			route := ws.DELETE(action.Path).To(local.HandleImagesAction(action.Verb, subAction, hasParams)).
				Filter(m).
				Doc(doc).
				Operation("deleteimages").
				Consumes(restful.MIME_XML, restful.MIME_JSON).
				Produces(restful.MIME_XML, restful.MIME_JSON)
			addParams(route, action.Params)
			ws.Route(route)
			break
		default:
			return fmt.Errorf("unsupported action")
		}
	}
	return nil
}

/*
	// HEAD
		"/containers/{name:.*}/archive"
	// GET
		"/containers/json"
		"/containers/{name:.*}/export"
		"/containers/{name:.*}/changes"
		"/containers/{name:.*}/json"
		"/containers/{name:.*}/top"
		"/containers/{name:.*}/logs"
		"/containers/{name:.*}/stats"
		"/containers/{name:.*}/attach/ws"
		"/containers/{name:.*}/archive"
	// POST
		"/containers/create"
		"/containers/{name:.*}/kill"
		"/containers/{name:.*}/pause"
		"/containers/{name:.*}/unpause"
		"/containers/{name:.*}/restart"
		"/containers/{name:.*}/start"
		"/containers/{name:.*}/stop"
		"/containers/{name:.*}/wait"
		"/containers/{name:.*}/resize"
		"/containers/{name:.*}/attach"
		"/containers/{name:.*}/copy"
		"/containers/{name:.*}/exec"
		"/exec/{name:.*}/start"
		"/exec/{name:.*}/resize"
		"/containers/{name:.*}/rename"
	// PUT
		"/containers/{name:.*}/archive"
	// DELETE
		"/containers/{name:.*}"
*/
func (a *APIInstaller) registerContainerHandlers(ws *restful.WebService) error {
	nameParam := ws.PathParameter("name", "name of the container").DataType("string")
	params := []*restful.Parameter{nameParam}
	actions := []action{}

	actions = append(actions, action{"GET", "/containers/json", nil})
	actions = append(actions, action{"GET", "/containers/{name}/export", params})
	actions = append(actions, action{"GET", "/containers/{name}/changes", params})
	actions = append(actions, action{"GET", "/containers/{name}/json", params})
	actions = append(actions, action{"GET", "/containers/{name}/top", params})
	actions = append(actions, action{"GET", "/containers/{name}/logs", params})
	actions = append(actions, action{"GET", "/containers/{name}/stats", params})
	actions = append(actions, action{"GET", "/containers/{name}/archive", params})

	actions = append(actions, action{"POST", "/containers/create", nil})
	actions = append(actions, action{"POST", "/containers/{name}/kill", params})
	actions = append(actions, action{"POST", "/containers/{name}/pause", params})
	actions = append(actions, action{"POST", "/containers/{name}/unpause", params})
	actions = append(actions, action{"POST", "/containers/{name}/restart", params})
	actions = append(actions, action{"POST", "/containers/{name}/start", params})
	actions = append(actions, action{"POST", "/containers/{name}/stop", params})
	actions = append(actions, action{"POST", "/containers/{name}/wait", params})
	actions = append(actions, action{"POST", "/containers/{name}/resize", params})
	actions = append(actions, action{"POST", "/containers/{name}/attach", params})
	actions = append(actions, action{"POST", "/containers/{name}/copy", params})
	actions = append(actions, action{"POST", "/containers/{name}/exec", params})
	actions = append(actions, action{"POST", "/containers/{name}/rename", params})

	actions = append(actions, action{"PUT", "/containers/{name}/archive", params})
	actions = append(actions, action{"DELETE", "/containers/{name}", params})

	for _, action := range actions {
		m := monitorFilter(action.Verb, "containers")
		hasParams := false
		if action.Params != nil {
			hasParams = true
		}
		fields := strings.Split(action.Path, "/")
		subAction := fields[len(fields)-1]
		switch action.Verb {
		case "GET":
			doc := "read the specified containers"
			route := ws.GET(action.Path).To(local.HandleContainersAction(action.Verb, subAction, hasParams)).
				Filter(m).
				Doc(doc).
				Operation("getcontainers"+subAction).
				Consumes(restful.MIME_XML, restful.MIME_JSON).
				Produces(restful.MIME_XML, restful.MIME_JSON)
			addParams(route, action.Params)
			ws.Route(route)
			break
		case "POST":
			doc := "update the specified containers"
			route := ws.POST(action.Path).To(local.HandleContainersAction(action.Verb, subAction, hasParams)).
				Filter(m).
				Doc(doc).
				Operation("postcontainers"+subAction).
				Consumes(restful.MIME_XML, restful.MIME_JSON).
				Produces(restful.MIME_XML, restful.MIME_JSON)
			addParams(route, action.Params)
			ws.Route(route)
			break
		case "PUT":
			doc := "put the specified containers"
			route := ws.PUT(action.Path).To(local.HandleContainersAction(action.Verb, subAction, hasParams)).
				Filter(m).
				Doc(doc).
				Operation("putcontainers").
				Consumes(restful.MIME_XML, restful.MIME_JSON).
				Produces(restful.MIME_XML, restful.MIME_JSON)
			addParams(route, action.Params)
			ws.Route(route)
			break
		case "DELETE":
			doc := "delete the specified containers"
			route := ws.DELETE(action.Path).To(local.HandleContainersAction(action.Verb, subAction, hasParams)).
				Filter(m).
				Doc(doc).
				Operation("deletecontainers").
				Consumes(restful.MIME_XML, restful.MIME_JSON).
				Produces(restful.MIME_XML, restful.MIME_JSON)
			addParams(route, action.Params)
			ws.Route(route)
			break
		default:
			return fmt.Errorf("unsupported action")
		}
	}
	return nil
}

// Wraps a http.Handler function inside a restful.RouteFunction
func routeFunction(handler http.Handler) restful.RouteFunction {
	return func(restReq *restful.Request, restResp *restful.Response) {
		handler.ServeHTTP(restResp.ResponseWriter, restReq.Request)
	}
}

func addParams(route *restful.RouteBuilder, params []*restful.Parameter) {
	for _, param := range params {
		route.Param(param)
	}
}

// TODO: this is incomplete, expand as needed.
// Convert the name of a golang type to the name of a JSON type
func typeToJSON(typeName string) string {
	switch typeName {
	case "bool", "*bool":
		return "boolean"
	case "uint8", "*uint8", "int", "*int", "int32", "*int32", "int64", "*int64", "uint32", "*uint32", "uint64", "*uint64":
		return "integer"
	case "float64", "*float64", "float32", "*float32":
		return "number"
	case "unversioned.Time", "*unversioned.Time":
		return "string"
	case "byte", "*byte":
		return "string"
	case "[]string", "[]*string":
		// TODO: Fix this when go-restful supports a way to specify an array query param:
		// https://github.com/emicklei/go-restful/issues/225
		return "string"
	default:
		return typeName
	}
}
