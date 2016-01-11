package httputils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/util/flushwriter"
	"k8s.io/kubernetes/pkg/util/wsstream"

	"github.com/golang/glog"
)

// write renders a returned runtime.Object to the response as a stream or an encoded object. If the object
// returned by the response implements rest.ResourceStreamer that interface will be used to render the
// response. The Accept header and current API version will be passed in, and the output will be copied
// directly to the response body. If content type is returned it is used, otherwise the content type will
// be "application/octet-stream". All other objects are sent to standard JSON serialization.
func write(statusCode int, groupVersion unversioned.GroupVersion, codec runtime.Codec, object runtime.Object, w http.ResponseWriter, req *http.Request) {
	if stream, ok := object.(rest.ResourceStreamer); ok {
		out, flush, contentType, err := stream.InputStream(groupVersion.String(), req.Header.Get("Accept"))
		if err != nil {
			errorJSONFatal(err, codec, w)
			return
		}
		if out == nil {
			// No output provided - return StatusNoContent
			w.WriteHeader(http.StatusNoContent)
			return
		}
		defer out.Close()

		if wsstream.IsWebSocketRequest(req) {
			r := wsstream.NewReader(out, true)
			if err := r.Copy(w, req); err != nil {
				util.HandleError(fmt.Errorf("error encountered while streaming results via websocket: %v", err))
			}
			return
		}

		if len(contentType) == 0 {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(statusCode)
		writer := w.(io.Writer)
		if flush {
			writer = flushwriter.Wrap(w)
		}
		io.Copy(writer, out)
		return
	}
	writeJSON(statusCode, codec, object, w, isPrettyPrint(req))
}

func isPrettyPrint(req *http.Request) bool {
	pp := req.URL.Query().Get("pretty")
	if len(pp) > 0 {
		pretty, _ := strconv.ParseBool(pp)
		return pretty
	}
	userAgent := req.UserAgent()
	// This covers basic all browers and cli http tools
	if strings.HasPrefix(userAgent, "curl") || strings.HasPrefix(userAgent, "Wget") || strings.HasPrefix(userAgent, "Mozilla/5.0") {
		return true
	}
	return false
}

// writeJSON renders an object as JSON to the response.
func writeJSON(statusCode int, codec runtime.Codec, object runtime.Object, w http.ResponseWriter, pretty bool) {
	w.Header().Set("Content-Type", "application/json")
	// We send the status code before we encode the object, so if we error, the status code stays but there will
	// still be an error object.  This seems ok, the alternative is to validate the object before
	// encoding, but this really should never happen, so it's wasted compute for every API request.
	w.WriteHeader(statusCode)
	if pretty {
		prettyJSON(codec, object, w)
		return
	}
	err := codec.EncodeToStream(object, w)
	if err != nil {
		errorJSONFatal(err, codec, w)
	}
}

func prettyJSON(codec runtime.Codec, object runtime.Object, w http.ResponseWriter) {
	formatted := &bytes.Buffer{}
	output, err := codec.Encode(object)
	if err != nil {
		errorJSONFatal(err, codec, w)
	}
	if err := json.Indent(formatted, output, "", "  "); err != nil {
		errorJSONFatal(err, codec, w)
		return
	}
	w.Write(formatted.Bytes())
}

// errorJSON renders an error to the response. Returns the HTTP status code of the error.
func ErrorJSON(err error, codec runtime.Codec, w http.ResponseWriter) int {
	status := errToAPIStatus(err)
	code := int(status.Code)
	writeJSON(code, codec, status, w, true)
	return code
}

// errorJSONFatal renders an error to the response, and if codec fails will render plaintext.
// Returns the HTTP status code of the error.
func errorJSONFatal(err error, codec runtime.Codec, w http.ResponseWriter) int {
	util.HandleError(fmt.Errorf("apiserver was unable to write a JSON response: %v", err))
	status := errToAPIStatus(err)
	code := int(status.Code)
	output, err := codec.Encode(status)
	if err != nil {
		w.WriteHeader(code)
		fmt.Fprintf(w, "%s: %s", status.Reason, status.Message)
		return code
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(output)
	return code
}

// writeRawJSON writes a non-API object in JSON.
func WriteRawJSON(statusCode int, object interface{}, w http.ResponseWriter) {
	output, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(output)
}

func parseTimeout(str string) time.Duration {
	if str != "" {
		timeout, err := time.ParseDuration(str)
		if err == nil {
			return timeout
		}
		glog.Errorf("Failed to parse %q: %v", str, err)
	}
	return 30 * time.Second
}

func readBody(req *http.Request) ([]byte, error) {
	defer req.Body.Close()
	return ioutil.ReadAll(req.Body)
}

// SplitPath returns the segments for a URL path.
func SplitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}
