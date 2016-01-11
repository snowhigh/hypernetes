package apiserver

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/conversion"
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

// An interface to see if an object supports swagger documentation as a method
type documentable interface {
	SwaggerDoc() map[string]string
}

// errEmptyName is returned when API requests do not fill the name section of the path.
var errEmptyName = errors.NewBadRequest("name must be provided")

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
		"/exec/{id:.*}/json"
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

// addObjectParams converts a runtime.Object into a set of go-restful Param() definitions on the route.
// The object must be a pointer to a struct; only fields at the top level of the struct that are not
// themselves interfaces or structs are used; only fields with a json tag that is non empty (the standard
// Go JSON behavior for omitting a field) become query parameters. The name of the query parameter is
// the JSON field name. If a description struct tag is set on the field, that description is used on the
// query parameter. In essence, it converts a standard JSON top level object into a query param schema.
func addObjectParams(ws *restful.WebService, route *restful.RouteBuilder, obj interface{}) error {
	sv, err := conversion.EnforcePtr(obj)
	if err != nil {
		return err
	}
	st := sv.Type()
	switch st.Kind() {
	case reflect.Struct:
		for i := 0; i < st.NumField(); i++ {
			name := st.Field(i).Name
			sf, ok := st.FieldByName(name)
			if !ok {
				continue
			}
			switch sf.Type.Kind() {
			case reflect.Interface, reflect.Struct:
			default:
				jsonTag := sf.Tag.Get("json")
				if len(jsonTag) == 0 {
					continue
				}
				jsonName := strings.SplitN(jsonTag, ",", 2)[0]
				if len(jsonName) == 0 {
					continue
				}

				var desc string
				if docable, ok := obj.(documentable); ok {
					desc = docable.SwaggerDoc()[jsonName]
				}
				route.Param(ws.QueryParameter(jsonName, desc).DataType(typeToJSON(sf.Type.String())))
			}
		}
	}
	return nil
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

// defaultStorageMetadata provides default answers to rest.StorageMetadata.
type defaultStorageMetadata struct{}

// defaultStorageMetadata implements rest.StorageMetadata
var _ rest.StorageMetadata = defaultStorageMetadata{}

func (defaultStorageMetadata) ProducesMIMETypes(verb string) []string {
	return nil
}
