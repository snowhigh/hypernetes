package master

import (
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/admission"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/latest"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/auth/authenticator"
	"k8s.io/kubernetes/pkg/auth/authorizer"
	"k8s.io/kubernetes/pkg/auth/handlers"
	"k8s.io/kubernetes/pkg/hypernetes/apiserver"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/storage"
	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/util/sets"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

// StorageDestinations is a mapping from API group & resource to
// the underlying storage interfaces.
type StorageDestinations struct {
	APIGroups map[string]*StorageDestinationsForAPIGroup
}

type StorageDestinationsForAPIGroup struct {
	Default   storage.Interface
	Overrides map[string]storage.Interface
}

func NewStorageDestinations() StorageDestinations {
	return StorageDestinations{
		APIGroups: map[string]*StorageDestinationsForAPIGroup{},
	}
}

func (s *StorageDestinations) AddAPIGroup(group string, defaultStorage storage.Interface) {
	s.APIGroups[group] = &StorageDestinationsForAPIGroup{
		Default:   defaultStorage,
		Overrides: map[string]storage.Interface{},
	}
}

func (s *StorageDestinations) AddStorageOverride(group, resource string, override storage.Interface) {
	if _, ok := s.APIGroups[group]; !ok {
		s.AddAPIGroup(group, nil)
	}
	if s.APIGroups[group].Overrides == nil {
		s.APIGroups[group].Overrides = map[string]storage.Interface{}
	}
	s.APIGroups[group].Overrides[resource] = override
}

func (s *StorageDestinations) get(group, resource string) storage.Interface {
	apigroup, ok := s.APIGroups[group]
	if !ok {
		glog.Errorf("No storage defined for API group: '%s'", apigroup)
		return nil
	}
	if apigroup.Overrides != nil {
		if client, exists := apigroup.Overrides[resource]; exists {
			return client
		}
	}
	return apigroup.Default
}

// Get all backends for all registered storage destinations.
// Used for getting all instances for health validations.
func (s *StorageDestinations) backends() []string {
	backends := sets.String{}
	for _, group := range s.APIGroups {
		if group.Default != nil {
			for _, backend := range group.Default.Backends(context.TODO()) {
				backends.Insert(backend)
			}
		}
		if group.Overrides != nil {
			for _, storage := range group.Overrides {
				for _, backend := range storage.Backends(context.TODO()) {
					backends.Insert(backend)
				}
			}
		}
	}
	return backends.List()
}

// Specifies the overrides for various API group versions.
// This can be used to enable/disable entire group versions or specific resources.
type APIGroupVersionOverride struct {
	// Whether to enable or disable this group version.
	Disable bool
	// List of overrides for individual resources in this group version.
	ResourceOverrides map[string]bool
}

// Config is a structure used to configure a Master.
type Config struct {
	StorageDestinations StorageDestinations
	// StorageVersions is a map between groups and their storage versions
	StorageVersions map[string]string
	EventTTL        time.Duration
	// allow downstream consumers to disable the core controller loops
	EnableCoreControllers bool
	EnableLogsSupport     bool
	// Allows api group versions or specific resources to be conditionally enabled/disabled.
	APIGroupVersionOverrides map[string]APIGroupVersionOverride
	// allow downstream consumers to disable the index route
	EnableIndex           bool
	EnableProfiling       bool
	EnableWatchCache      bool
	APIPrefix             string
	CorsAllowedOriginList []string
	Authenticator         authenticator.Request
	Authorizer            authorizer.Authorizer
	AdmissionControl      admission.Interface

	// Map requests to contexts. Exported so downstream consumers can provider their own mappers
	RequestContextMapper api.RequestContextMapper

	// If specified, requests will be allocated a random timeout between this value, and twice this value.
	// Note that it is up to the request handlers to ignore or honor this timeout. In seconds.
	MinRequestTimeout int

	// Number of masters running; all masters must be started with the
	// same value for this field. (Numbers > 1 currently untested.)
	MasterCount int

	// The port on PublicAddress where a read-write server will be installed.
	// Defaults to 6443 if not set.
	ReadWritePort int

	// ExternalHost is the host name to use for external (public internet) facing URLs (e.g. Swagger)
	ExternalHost string

	// PublicAddress is the IP address where members of the cluster (kubelet,
	// kube-proxy, services, etc.) can reach the master.
	// If nil or 0.0.0.0, the host's default interface will be used.
	PublicAddress net.IP

	// Control the interval that pod, node IP, and node heath status caches
	// expire.
	CacheTimeout time.Duration

	KubernetesServiceNodePort int
}

func (c *Config) storageDecorator() generic.StorageDecorator {
	return generic.UndecoratedStorage
}

type InstallSSHKey func(user string, data []byte) error

// Master contains state for a Kubernetes cluster master/api server.
type Master struct {
	// "Inputs", Copied from Config
	cacheTimeout      time.Duration
	minRequestTimeout time.Duration

	mux                      apiserver.Mux
	muxHelper                *apiserver.MuxHelper
	handlerContainer         *restful.Container
	rootWebService           *restful.WebService
	enableCoreControllers    bool
	enableLogsSupport        bool
	enableProfiling          bool
	enableWatchCache         bool
	apiPrefix                string
	corsAllowedOriginList    []string
	authenticator            authenticator.Request
	authorizer               authorizer.Authorizer
	admissionControl         admission.Interface
	masterCount              int
	apiGroupVersionOverrides map[string]APIGroupVersionOverride
	requestContextMapper     api.RequestContextMapper

	// External host is the name that should be used in external (public internet) URLs for this master
	externalHost string
	// clusterIP is the IP address of the master within the cluster.
	clusterIP            net.IP
	publicReadWritePort  int
	serviceReadWriteIP   net.IP
	serviceReadWritePort int
	masterServices       *util.Runner
	extraServicePorts    []api.ServicePort
	extraEndpointPorts   []api.EndpointPort

	// storage contains the RESTful endpoints exposed by this master
	storage map[string]rest.Storage

	// "Outputs"
	Handler         http.Handler
	InsecureHandler http.Handler

	KubernetesServiceNodePort int
}

// setDefaults fills in any fields not set that are required to have valid data.
func setDefaults(c *Config) {
	if c.MasterCount == 0 {
		// Clearly, there will be at least one master.
		c.MasterCount = 1
	}
	if c.ReadWritePort == 0 {
		c.ReadWritePort = 6443
	}
	if c.CacheTimeout == 0 {
		c.CacheTimeout = 5 * time.Second
	}
	if c.RequestContextMapper == nil {
		c.RequestContextMapper = api.NewRequestContextMapper()
	}
}

// New returns a new instance of Master from the given config.
// Certain config fields will be set to a default value if unset,
// including:
//   ServiceClusterIPRange
//   ServiceNodePortRange
//   MasterCount
//   ReadWritePort
//   PublicAddress
// Certain config fields must be specified, including:
// Public fields:
//   Handler -- The returned master has a field TopHandler which is an
//   http.Handler which handles all the endpoints provided by the master,
//   including the API, the UI, and miscellaneous debugging endpoints.  All
//   these are subject to authorization and authentication.
//   InsecureHandler -- an http.Handler which handles all the same
//   endpoints as Handler, but no authorization and authentication is done.
// Public methods:
//   HandleWithAuth -- Allows caller to add an http.Handler for an endpoint
//   that uses the same authentication and authorization (if any is configured)
//   as the master's built-in endpoints.
//   If the caller wants to add additional endpoints not using the master's
//   auth, then the caller should create a handler for those endpoints, which delegates the
//   any unhandled paths to "Handler".
func New(c *Config) *Master {
	setDefaults(c)

	m := &Master{
		rootWebService:           new(restful.WebService),
		enableCoreControllers:    c.EnableCoreControllers,
		enableLogsSupport:        c.EnableLogsSupport,
		enableProfiling:          c.EnableProfiling,
		enableWatchCache:         c.EnableWatchCache,
		apiPrefix:                c.APIPrefix,
		corsAllowedOriginList:    c.CorsAllowedOriginList,
		authenticator:            c.Authenticator,
		authorizer:               c.Authorizer,
		admissionControl:         c.AdmissionControl,
		apiGroupVersionOverrides: c.APIGroupVersionOverrides,
		requestContextMapper:     c.RequestContextMapper,

		cacheTimeout:      c.CacheTimeout,
		minRequestTimeout: time.Duration(c.MinRequestTimeout) * time.Second,

		masterCount:         c.MasterCount,
		externalHost:        c.ExternalHost,
		clusterIP:           c.PublicAddress,
		publicReadWritePort: c.ReadWritePort,
		// TODO: serviceReadWritePort should be passed in as an argument, it may not always be 443
		serviceReadWritePort: 443,

		KubernetesServiceNodePort: c.KubernetesServiceNodePort,
	}

	var handlerContainer *restful.Container
	mux := http.NewServeMux()
	m.mux = mux
	handlerContainer = NewHandlerContainer(mux)
	m.handlerContainer = handlerContainer
	// Use CurlyRouter to be able to use regular expressions in paths. Regular expressions are required in paths for example for proxy (where the path is proxy/{kind}/{name}/{*})
	m.handlerContainer.Router(restful.CurlyRouter{})
	m.muxHelper = &apiserver.MuxHelper{Mux: m.mux, RegisteredPaths: []string{}}

	m.init(c)

	return m
}

// HandleWithAuth adds an http.Handler for pattern to an http.ServeMux
// Applies the same authentication and authorization (if any is configured)
// to the request is used for the master's built-in endpoints.
func (m *Master) HandleWithAuth(pattern string, handler http.Handler) {
	// TODO: Add a way for plugged-in endpoints to translate their
	// URLs into attributes that an Authorizer can understand, and have
	// sensible policy defaults for plugged-in endpoints.  This will be different
	// for generic endpoints versus REST object endpoints.
	// TODO: convert to go-restful
	m.muxHelper.Handle(pattern, handler)
}

// HandleFuncWithAuth adds an http.Handler for pattern to an http.ServeMux
// Applies the same authentication and authorization (if any is configured)
// to the request is used for the master's built-in endpoints.
func (m *Master) HandleFuncWithAuth(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	// TODO: convert to go-restful
	m.muxHelper.HandleFunc(pattern, handler)
}

func NewHandlerContainer(mux *http.ServeMux) *restful.Container {
	container := restful.NewContainer()
	container.ServeMux = mux
	apiserver.InstallRecoverHandler(container)
	return container
}

// init initializes master.
func (m *Master) init(c *Config) {

	apiVersions := []string{}
	// Install v1 unless disabled.
	if !m.apiGroupVersionOverrides["hapi/v1"].Disable {
		if err := m.api_v1().InstallREST(m.handlerContainer); err != nil {
			glog.Fatalf("Unable to setup API v1: %v", err)
		}
		apiVersions = append(apiVersions, "v1")
	}

	apiserver.AddApiWebService(m.handlerContainer, c.APIPrefix, apiVersions)

	// Register root handler.
	// We do not register this using restful Webservice since we do not want to surface this in api docs.
	// Allow master to be embedded in contexts which already have something registered at the root
	if c.EnableIndex {
		m.mux.HandleFunc("/", apiserver.IndexHandler(m.handlerContainer, m.muxHelper))
	}

	if c.EnableLogsSupport {
		apiserver.InstallLogsSupport(m.muxHelper)
	}

	if c.EnableProfiling {
		m.mux.HandleFunc("/debug/pprof/", pprof.Index)
		m.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		m.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}

	handler := http.Handler(m.mux.(*http.ServeMux))
	insecureHandler := handler

	// TODO: handle CORS and auth using go-restful
	// See github.com/emicklei/go-restful/blob/master/examples/restful-CORS-filter.go, and
	// github.com/emicklei/go-restful/blob/master/examples/restful-basic-authentication.go

	attributeGetter := apiserver.NewRequestAttributeGetter(m.requestContextMapper, m.newRequestInfoResolver())
	handler = apiserver.WithAuthorizationCheck(handler, attributeGetter, m.authorizer)

	// Install Authenticator
	if c.Authenticator != nil {
		authenticatedHandler, err := handlers.NewRequestAuthenticator(m.requestContextMapper, c.Authenticator, handlers.Unauthorized(false), handler)
		if err != nil {
			glog.Fatalf("Could not initialize authenticator: %v", err)
		}
		handler = authenticatedHandler
	}

	// Since OPTIONS request cannot carry authn headers (by w3c standards), we are doing CORS check
	// before auth check. Otherwise all the CORS request will be rejected.
	if len(c.CorsAllowedOriginList) > 0 {
		allowedOriginRegexps, err := util.CompileRegexps(c.CorsAllowedOriginList)
		if err != nil {
			glog.Fatalf("Invalid CORS allowed origin, --cors-allowed-origins flag was set to %v - %v", strings.Join(c.CorsAllowedOriginList, ","), err)
		}
		handler = apiserver.CORS(handler, allowedOriginRegexps, nil, nil, "true")
		insecureHandler = apiserver.CORS(insecureHandler, allowedOriginRegexps, nil, nil, "true")
	}

	m.InsecureHandler = insecureHandler

	// Install root web services
	//m.handlerContainer.Add(m.rootWebService)

	// TODO: Make this optional?  Consumers of master depend on this currently.
	m.Handler = handler

	// After all wrapping is done, put a context filter around both handlers
	if handler, err := api.NewRequestContextFilter(m.requestContextMapper, m.Handler); err != nil {
		glog.Fatalf("Could not initialize request context filter: %v", err)
	} else {
		m.Handler = handler
	}

	if handler, err := api.NewRequestContextFilter(m.requestContextMapper, m.InsecureHandler); err != nil {
		glog.Fatalf("Could not initialize request context filter: %v", err)
	} else {
		m.InsecureHandler = handler
	}
}

func (m *Master) newRequestInfoResolver() *apiserver.RequestInfoResolver {
	return &apiserver.RequestInfoResolver{
		sets.NewString(strings.Trim(m.apiPrefix, "/"), strings.Trim("test", "/")), // all possible API prefixes
		sets.NewString(strings.Trim(m.apiPrefix, "/")),                            // APIPrefixes that won't have groups (legacy)
	}
}

func (m *Master) defaultAPIGroupVersion() *apiserver.APIGroupVersion {
	return &apiserver.APIGroupVersion{
		Root:                m.apiPrefix,
		RequestInfoResolver: m.newRequestInfoResolver(),

		Mapper: latest.GroupOrDie("").RESTMapper,

		Creater:   api.Scheme,
		Convertor: api.Scheme,
		Typer:     api.Scheme,
		Linker:    latest.GroupOrDie("").SelfLinker,

		Admit:   m.admissionControl,
		Context: m.requestContextMapper,

		MinRequestTimeout: m.minRequestTimeout,
	}
}

// api_v1 returns the resources and codec for API version v1.
func (m *Master) api_v1() *apiserver.APIGroupVersion {
	storage := make(map[string]rest.Storage)
	for k, v := range m.storage {
		storage[strings.ToLower(k)] = v
	}
	version := m.defaultAPIGroupVersion()
	version.Storage = storage
	version.GroupVersion = unversioned.GroupVersion{Version: "v1"}
	version.Codec = v1.Codec
	return version
}
