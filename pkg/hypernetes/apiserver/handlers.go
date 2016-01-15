/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apiserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/httplog"
	"k8s.io/kubernetes/pkg/hypernetes/auth/authorizer"
	"k8s.io/kubernetes/pkg/hypernetes/httputils"
	"k8s.io/kubernetes/pkg/util/sets"
)

// Constant for the retry-after interval on rate limiting.
// TODO: maybe make this dynamic? or user-adjustable?
const RetryAfter = "1"

// IsReadOnlyReq() is true for any (or at least many) request which has no observable
// side effects on state of apiserver (though there may be internal side effects like
// caching and logging).
func IsReadOnlyReq(req http.Request) bool {
	if req.Method == "GET" {
		// TODO: add OPTIONS and HEAD if we ever support those.
		return true
	}
	return false
}

// ReadOnly passes all GET requests on to handler, and returns an error on all other requests.
func ReadOnly(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if IsReadOnlyReq(*req) {
			handler.ServeHTTP(w, req)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "This is a read-only endpoint.")
	})
}

// MaxInFlight limits the number of in-flight requests to buffer size of the passed in channel.
func MaxInFlightLimit(c chan bool, longRunningRequestRE *regexp.Regexp, handler http.Handler) http.Handler {
	if c == nil {
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if longRunningRequestRE.MatchString(r.URL.Path) {
			// Skip tracking long running events.
			handler.ServeHTTP(w, r)
			return
		}
		select {
		case c <- true:
			defer func() { <-c }()
			handler.ServeHTTP(w, r)
		default:
			tooManyRequests(w)
		}
	})
}

func tooManyRequests(w http.ResponseWriter) {
	// Return a 429 status indicating "Too Many Requests"
	w.Header().Set("Retry-After", RetryAfter)
	http.Error(w, "Too many requests, please try again later.", errors.StatusTooManyRequests)
}

// RecoverPanics wraps an http Handler to recover and log panics.
func RecoverPanics(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				http.Error(w, "apis panic. Look in log for details.", http.StatusInternalServerError)
				glog.Errorf("APIServer panic'd on %v %v: %v\n%s\n", req.Method, req.RequestURI, x, debug.Stack())
			}
		}()
		defer httplog.NewLogged(req, &w).StacktraceWhen(
			httplog.StatusIsNot(
				http.StatusOK,
				http.StatusCreated,
				http.StatusAccepted,
				http.StatusBadRequest,
				http.StatusMovedPermanently,
				http.StatusTemporaryRedirect,
				http.StatusConflict,
				http.StatusNotFound,
				http.StatusUnauthorized,
				http.StatusForbidden,
				errors.StatusUnprocessableEntity,
				http.StatusSwitchingProtocols,
			),
		).Log()

		// Dispatch to the internal handler
		handler.ServeHTTP(w, req)
	})
}

// TimeoutHandler returns an http.Handler that runs h with a timeout
// determined by timeoutFunc. The new http.Handler calls h.ServeHTTP to handle
// each request, but if a call runs for longer than its time limit, the
// handler responds with a 503 Service Unavailable error and the message
// provided. (If msg is empty, a suitable default message with be sent.) After
// the handler times out, writes by h to its http.ResponseWriter will return
// http.ErrHandlerTimeout. If timeoutFunc returns a nil timeout channel, no
// timeout will be enforced.
func TimeoutHandler(h http.Handler, timeoutFunc func(*http.Request) (timeout <-chan time.Time, msg string)) http.Handler {
	return &timeoutHandler{h, timeoutFunc}
}

type timeoutHandler struct {
	handler http.Handler
	timeout func(*http.Request) (<-chan time.Time, string)
}

func (t *timeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	after, msg := t.timeout(r)
	if after == nil {
		t.handler.ServeHTTP(w, r)
		return
	}

	done := make(chan struct{}, 1)
	tw := newTimeoutWriter(w)
	go func() {
		t.handler.ServeHTTP(tw, r)
		done <- struct{}{}
	}()
	select {
	case <-done:
		return
	case <-after:
		tw.timeout(msg)
	}
}

type timeoutWriter interface {
	http.ResponseWriter
	timeout(string)
}

func newTimeoutWriter(w http.ResponseWriter) timeoutWriter {
	base := &baseTimeoutWriter{w: w}

	_, notifiable := w.(http.CloseNotifier)
	_, hijackable := w.(http.Hijacker)

	switch {
	case notifiable && hijackable:
		return &closeHijackTimeoutWriter{base}
	case notifiable:
		return &closeTimeoutWriter{base}
	case hijackable:
		return &hijackTimeoutWriter{base}
	default:
		return base
	}
}

type baseTimeoutWriter struct {
	w http.ResponseWriter

	mu          sync.Mutex
	timedOut    bool
	wroteHeader bool
	hijacked    bool
}

func (tw *baseTimeoutWriter) Header() http.Header {
	return tw.w.Header()
}

func (tw *baseTimeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.wroteHeader = true
	if tw.hijacked {
		return 0, http.ErrHijacked
	}
	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	return tw.w.Write(p)
}

func (tw *baseTimeoutWriter) Flush() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if flusher, ok := tw.w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (tw *baseTimeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.wroteHeader || tw.hijacked {
		return
	}
	tw.wroteHeader = true
	tw.w.WriteHeader(code)
}

func (tw *baseTimeoutWriter) timeout(msg string) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.wroteHeader && !tw.hijacked {
		tw.w.WriteHeader(http.StatusGatewayTimeout)
		if msg != "" {
			tw.w.Write([]byte(msg))
		} else {
			enc := json.NewEncoder(tw.w)
			enc.Encode(errors.NewServerTimeout("", "", 0))
		}
	}
	tw.timedOut = true
}

func (tw *baseTimeoutWriter) closeNotify() <-chan bool {
	return tw.w.(http.CloseNotifier).CloseNotify()
}

func (tw *baseTimeoutWriter) hijack() (net.Conn, *bufio.ReadWriter, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return nil, nil, http.ErrHandlerTimeout
	}
	conn, rw, err := tw.w.(http.Hijacker).Hijack()
	if err == nil {
		tw.hijacked = true
	}
	return conn, rw, err
}

type closeTimeoutWriter struct {
	*baseTimeoutWriter
}

func (tw *closeTimeoutWriter) CloseNotify() <-chan bool {
	return tw.closeNotify()
}

type hijackTimeoutWriter struct {
	*baseTimeoutWriter
}

func (tw *hijackTimeoutWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return tw.hijack()
}

type closeHijackTimeoutWriter struct {
	*baseTimeoutWriter
}

func (tw *closeHijackTimeoutWriter) CloseNotify() <-chan bool {
	return tw.closeNotify()
}

func (tw *closeHijackTimeoutWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return tw.hijack()
}

// TODO: use restful.CrossOriginResourceSharing
// Simple CORS implementation that wraps an http Handler
// For a more detailed implementation use https://github.com/martini-contrib/cors
// or implement CORS at your proxy layer
// Pass nil for allowedMethods and allowedHeaders to use the defaults
func CORS(handler http.Handler, allowedOriginPatterns []*regexp.Regexp, allowedMethods []string, allowedHeaders []string, allowCredentials string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, pattern := range allowedOriginPatterns {
				if allowed = pattern.MatchString(origin); allowed {
					break
				}
			}
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				// Set defaults for methods and headers if nothing was passed
				if allowedMethods == nil {
					allowedMethods = []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"}
				}
				if allowedHeaders == nil {
					allowedHeaders = []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "X-Requested-With", "If-Modified-Since"}
				}
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
				w.Header().Set("Access-Control-Allow-Credentials", allowCredentials)

				// Stop here if its a preflight OPTIONS request
				if req.Method == "OPTIONS" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
		}
		// Dispatch to the next handler
		handler.ServeHTTP(w, req)
	})
}

// RequestAttributeGetter is a function that extracts authorizer.Attributes from an http.Request
type RequestAttributeGetter interface {
	GetAttribs(req *http.Request) (attribs authorizer.Attributes)
}

type requestAttributeGetter struct {
	requestContextMapper api.RequestContextMapper
	requestInfoResolver  *RequestInfoResolver
}

// NewAttributeGetter returns an object which implements the RequestAttributeGetter interface.
func NewRequestAttributeGetter(requestContextMapper api.RequestContextMapper, requestInfoResolver *RequestInfoResolver) RequestAttributeGetter {
	return &requestAttributeGetter{requestContextMapper, requestInfoResolver}
}

func (r *requestAttributeGetter) GetAttribs(req *http.Request) authorizer.Attributes {
	attribs := authorizer.AttributesRecord{}

	ctx, ok := r.requestContextMapper.Get(req)
	if ok {
		user, ok := api.UserFrom(ctx)
		if ok {
			attribs.User = user
		}
	}

	requestInfo, _ := r.requestInfoResolver.GetRequestInfo(req)

	// Start with common attributes that apply to resource and non-resource requests
	attribs.ResourceRequest = requestInfo.IsResourceRequest
	attribs.Path = requestInfo.Path
	attribs.Action = requestInfo.Action
	attribs.Verb = requestInfo.Verb

	// If the request was for a resource in an API group, include that info
	attribs.APIGroup = requestInfo.APIGroup

	// If a path follows the conventions of the REST object store, then
	// we can extract the resource.  Otherwise, not.
	attribs.Resource = requestInfo.Resource
	attribs.Name = requestInfo.Name

	return &attribs
}

// WithAuthorizationCheck passes all authorized requests on to handler, and returns a forbidden error otherwise.
func WithAuthorizationCheck(handler http.Handler, getAttribs RequestAttributeGetter, a authorizer.Authorizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := a.Authorize(getAttribs.GetAttribs(req))
		if err == nil {
			handler.ServeHTTP(w, req)
			return
		}
		httputils.Forbidden(w, req)
	})
}

// RequestInfo holds information parsed from the http.Request
type RequestInfo struct {
	// IsResourceRequest indicates whether or not the request is for an API resource or subresource
	IsResourceRequest bool
	// Path is the URL path of the request
	Path   string
	Verb   string
	Action string

	APIPrefix  string
	APIGroup   string
	APIVersion string
	// Resource is the name of the resource being requested.  This is not the kind.  For example: containers, images
	Resource string
	// Name is empty for some verbs, but if the request directly indicates a name (not in body content) then this field is filled in.
	Name string
}

type RequestInfoResolver struct {
	APIPrefixes          sets.String
	GrouplessAPIPrefixes sets.String
}

// GetRequestInfo returns the information from the http request.  If error is not nil, RequestInfo holds the information as best it is known before the failure
// It handles both resource and non-resource requests and fills in all the pertinent information for each.
// Valid Inputs:
// Resource paths without action
// /hapi/{version}/info
// /hapi/{version}/version
// /hapi/{version}/volumes
// /hapi/{version}/networks
//
// Resource paths with action
// /hapi/{version}/images/json
// /hapi/{version}/images/{name}/json
// /hapi/{version}/images/{name}/history
// /hapi/{version}/images/{name}/list
// /hapi/{version}/containers/json
// /hapi/{version}/containers/{name}/json
// /hapi/{version}/{resource}/{resourceName}/{action}
// /
func (r *RequestInfoResolver) GetRequestInfo(req *http.Request) (RequestInfo, error) {
	// start with a non-resource request until proven otherwise
	requestInfo := RequestInfo{
		IsResourceRequest: false,
		Path:              req.URL.Path,
		Verb:              req.Method,
	}

	currentParts := httputils.SplitPath(req.URL.Path)
	if len(currentParts) < 3 {
		// return a non-resource request
		return requestInfo, nil
	}

	if !r.APIPrefixes.Has(currentParts[0]) {
		// return a non-resource request
		return requestInfo, nil
	}
	requestInfo.APIPrefix = currentParts[0]
	currentParts = currentParts[1:]

	if !r.GrouplessAPIPrefixes.Has(requestInfo.APIPrefix) {
		// one part (APIPrefix) has already been consumed, so this is actually "do we have four parts?"
		if len(currentParts) < 3 {
			// return a non-resource request
			return requestInfo, nil
		}

		requestInfo.APIGroup = currentParts[0]
		currentParts = currentParts[1:]
	}

	requestInfo.IsResourceRequest = true
	requestInfo.APIVersion = currentParts[0]
	currentParts = currentParts[1:]

	length := len(currentParts)
	if length == 1 {
		if currentParts[0] == "networks" || currentParts[0] == "volumes" {
			requestInfo.Resource = currentParts[0]
		} else {
			requestInfo.Action = currentParts[0]
		}
	} else {
		requestInfo.Resource = currentParts[0]
		if length == 2 {
			requestInfo.Action = currentParts[1]
			if req.Method == "DELETE" {
				requestInfo.Action = "delete"
				requestInfo.Name = currentParts[1]
			} else if currentParts[1] == "networks" || currentParts[1] == "volumes" {
				// GET /networks/{id}
				// GET /volumes/{name}
				requestInfo.Action = "get"
				requestInfo.Name = currentParts[1]
			}
		}
		if length == 3 {
			requestInfo.Name = currentParts[1]
			requestInfo.Action = currentParts[2]
		}
	}

	return requestInfo, nil
}
