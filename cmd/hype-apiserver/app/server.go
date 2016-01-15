// Package app does all of the work necessary to create a Kubernetes
// APIServer by binding together the API, master and APIServer infrastructure.
// It can be configured and called directly or via the hyperkube framework.
package app

import (
	"crypto/tls"
	"net"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"time"

	systemd "github.com/coreos/go-systemd/daemon"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	hapiserver "k8s.io/kubernetes/pkg/hypernetes/apiserver"
	hmaster "k8s.io/kubernetes/pkg/hypernetes/master"
	"k8s.io/kubernetes/pkg/hypernetes/storage"
	"k8s.io/kubernetes/pkg/hypernetes/storage/mongodb"
	"k8s.io/kubernetes/pkg/util"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	// Maximum duration before timing out read/write requests
	// Set to a value larger than the timeouts in each watch server.
	ReadWriteTimeout = time.Minute * 60
	//TODO: This can be tightened up. It still matches objects named watch or proxy.
	defaultLongRunningRequestRE = "(/|^)((logs?|portforward|exec|attach)/?$)"
)

// APIServer runs a hypernetes api server.
type APIServer struct {
	InsecureBindAddress  net.IP
	InsecurePort         int
	BindAddress          net.IP
	AdvertiseAddress     net.IP
	SecurePort           int
	TLSCertFile          string
	TLSPrivateKeyFile    string
	CertDirectory        string
	APIPrefix            string
	KeystoneURL          string
	MongodbURL           string
	MaxRequestsInFlight  int
	MinRequestTimeout    int
	LongRunningRequestRE string
}

// NewAPIServer creates a new APIServer object with default parameters
func NewAPIServer() *APIServer {
	s := APIServer{
		InsecurePort:        8888,
		InsecureBindAddress: net.ParseIP("127.0.0.1"),
		BindAddress:         net.ParseIP("0.0.0.0"),
		MongodbURL:          "127.0.0.1",
		SecurePort:          6443,
		APIPrefix:           "/hapi",
	}

	return &s
}

// NewAPIServerCommand creates a *cobra.Command object with default parameters
func NewAPIServerCommand() *cobra.Command {
	s := NewAPIServer()
	s.AddFlags(pflag.CommandLine)
	cmd := &cobra.Command{
		Use: "hype-apiserver",
		Long: `The Hypernetes API server validates and configures data
for the api objects which include pods, services, replicationcontrollers, and
others. The API Server services REST operations and provides the frontend to the
cluster's shared state through which all other components interact.`,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	return cmd
}

// AddFlags adds flags for a specific APIServer to the specified FlagSet
func (s *APIServer) AddFlags(fs *pflag.FlagSet) {
	// Note: the weird ""+ in below lines seems to be the only way to get gofmt to
	// arrange these text blocks sensibly. Grrr.
	fs.IntVar(&s.InsecurePort, "insecure-port", s.InsecurePort, ""+
		"The port on which to serve unsecured, unauthenticated access. Default 8888. It is assumed "+
		"that firewall rules are set up such that this port is not reachable from outside of "+
		"the cluster and that port 443 on the cluster's public address is proxied to this "+
		"port. This is performed by nginx in the default setup.")
	fs.IntVar(&s.InsecurePort, "port", s.InsecurePort, "DEPRECATED: see --insecure-port instead")
	fs.MarkDeprecated("port", "see --insecure-port instead")
	fs.IPVar(&s.InsecureBindAddress, "insecure-bind-address", s.InsecureBindAddress, ""+
		"The IP address on which to serve the --insecure-port (set to 0.0.0.0 for all interfaces). "+
		"Defaults to localhost.")
	fs.IPVar(&s.InsecureBindAddress, "address", s.InsecureBindAddress, "DEPRECATED: see --insecure-bind-address instead")
	fs.MarkDeprecated("address", "see --insecure-bind-address instead")
	fs.IPVar(&s.BindAddress, "bind-address", s.BindAddress, ""+
		"The IP address on which to listen for the --secure-port port. The "+
		"associated interface(s) must be reachable by the rest of the cluster, and by CLI/web "+
		"clients. If blank, all interfaces will be used (0.0.0.0).")
	fs.IPVar(&s.AdvertiseAddress, "advertise-address", s.AdvertiseAddress, ""+
		"The IP address on which to advertise the apiserver to members of the cluster. This "+
		"address must be reachable by the rest of the cluster. If blank, the --bind-address "+
		"will be used. If --bind-address is unspecified, the host's default interface will "+
		"be used.")
	fs.IPVar(&s.BindAddress, "public-address-override", s.BindAddress, "DEPRECATED: see --bind-address instead")
	fs.MarkDeprecated("public-address-override", "see --bind-address instead")
	fs.IntVar(&s.SecurePort, "secure-port", s.SecurePort, ""+
		"The port on which to serve HTTPS with authentication and authorization. If 0, "+
		"don't serve HTTPS at all.")
	fs.StringVar(&s.TLSCertFile, "tls-cert-file", s.TLSCertFile, ""+
		"File containing x509 Certificate for HTTPS.  (CA cert, if any, concatenated after server cert). "+
		"If HTTPS serving is enabled, and --tls-cert-file and --tls-private-key-file are not provided, "+
		"a self-signed certificate and key are generated for the public address and saved to /var/run/kubernetes.")
	fs.StringVar(&s.TLSPrivateKeyFile, "tls-private-key-file", s.TLSPrivateKeyFile, "File containing x509 private key matching --tls-cert-file.")
	fs.StringVar(&s.CertDirectory, "cert-dir", s.CertDirectory, "The directory where the TLS certs are located (by default /var/run/kubernetes). "+
		"If --tls-cert-file and --tls-private-key-file are provided, this flag will be ignored.")
	fs.StringVar(&s.APIPrefix, "api-prefix", s.APIPrefix, "The prefix for API requests on the server. Default '/api'.")
	fs.MarkDeprecated("api-prefix", "--api-prefix is deprecated and will be removed when the v1 API is retired.")
	fs.StringVar(&s.KeystoneURL, "keystone-url", s.KeystoneURL, "If passed, activates the keystone authentication plugin")
	fs.StringVar(&s.MongodbURL, "mongodb-url", s.MongodbURL, "Hype-APIServer backend storage URL")
	fs.IntVar(&s.MaxRequestsInFlight, "max-requests-inflight", 400, "The maximum number of requests in flight at a given time.  When the server exceeds this, it rejects requests.  Zero for no limit.")
	fs.IntVar(&s.MinRequestTimeout, "min-request-timeout", 1800, "An optional field indicating the minimum number of seconds a handler must keep a request open before timing it out. Currently only honored by the watch request handler, which picks a randomized value above this number as the connection timeout, to spread out load.")
	fs.StringVar(&s.LongRunningRequestRE, "long-running-request-regexp", defaultLongRunningRequestRE, "A regular expression matching long running requests which should be excluded from maximum inflight request handling.")
}

// Run runs the specified APIServer.  This should never exit.
func (s *APIServer) Run(_ []string) error {
	var mongodbStorage storage.Interface

	clientConfig := &client.Config{
		Host: net.JoinHostPort(s.InsecureBindAddress.String(), strconv.Itoa(s.InsecurePort)),
	}
	client, err := client.New(clientConfig)
	if err != nil {
		glog.Fatalf("Invalid server address: %v", err)
	}
	_ = client

	mongodbStorage, err = mongodb.NewMongodbStorage(s.MongodbURL)
	if err != nil {
		return err
	}

	authenticator, err := hapiserver.NewAuthenticator(hapiserver.AuthenticatorConfig{
		Storage:     mongodbStorage,
		KeystoneURL: s.KeystoneURL,
	})

	if err != nil {
		glog.Fatalf("Invalid Authentication Config: %v", err)
	}

	authorizer, err := hapiserver.NewAuthorizerFromAuthorizationConfig(mongodbStorage)
	if err != nil {
		glog.Fatalf("Invalid Authorization Config: %v", err)
	}

	config := &hmaster.Config{
		EnableCoreControllers: true,
		EnableProfiling:       true,
		EnableWatchCache:      true,
		APIPrefix:             s.APIPrefix,
		ReadWritePort:         s.SecurePort,
		PublicAddress:         s.AdvertiseAddress,
		Authenticator:         authenticator,
		Authorizer:            authorizer,
		Storage:               mongodbStorage,
	}
	m := hmaster.New(config)

	// We serve on 2 ports.  See docs/accessing_the_api.md
	secureLocation := ""
	if s.SecurePort != 0 {
		secureLocation = net.JoinHostPort(s.BindAddress.String(), strconv.Itoa(s.SecurePort))
	}
	insecureLocation := net.JoinHostPort(s.InsecureBindAddress.String(), strconv.Itoa(s.InsecurePort))

	// See the flag commentary to understand our assumptions when opening the read-only and read-write ports.
	var sem chan bool
	if s.MaxRequestsInFlight > 0 {
		sem = make(chan bool, s.MaxRequestsInFlight)
	}

	longRunningRE := regexp.MustCompile(s.LongRunningRequestRE)
	longRunningTimeout := func(req *http.Request) (<-chan time.Time, string) {
		// TODO unify this with apiserver.MaxInFlightLimit
		if longRunningRE.MatchString(req.URL.Path) || req.URL.Query().Get("watch") == "true" {
			return nil, ""
		}
		return time.After(time.Minute), ""
	}

	if secureLocation != "" {
		handler := hapiserver.TimeoutHandler(m.Handler, longRunningTimeout)
		secureServer := &http.Server{
			Addr:           secureLocation,
			Handler:        hapiserver.MaxInFlightLimit(sem, longRunningRE, hapiserver.RecoverPanics(handler)),
			MaxHeaderBytes: 1 << 20,
			TLSConfig: &tls.Config{
				// Change default from SSLv3 to TLSv1.0 (because of POODLE vulnerability)
				MinVersion: tls.VersionTLS10,
			},
		}

		glog.Infof("Serving securely on %s", secureLocation)
		if s.TLSCertFile == "" && s.TLSPrivateKeyFile == "" {
			s.TLSCertFile = path.Join(s.CertDirectory, "apiserver.crt")
			s.TLSPrivateKeyFile = path.Join(s.CertDirectory, "apiserver.key")
			// TODO (cjcullen): Is PublicAddress the right address to sign a cert with?
			alternateIPs := []net.IP{net.ParseIP("127.0.0.1")}
			alternateDNS := []string{"kubernetes.default.svc", "kubernetes.default", "kubernetes"}
			// It would be nice to set a fqdn subject alt name, but only the kubelets know, the apiserver is clueless
			// alternateDNS = append(alternateDNS, "kubernetes.default.svc.CLUSTER.DNS.NAME")
			if err := util.GenerateSelfSignedCert(config.PublicAddress.String(), s.TLSCertFile, s.TLSPrivateKeyFile, alternateIPs, alternateDNS); err != nil {
				glog.Errorf("Unable to generate self signed cert: %v", err)
			} else {
				glog.Infof("Using self-signed cert (%s, %s)", s.TLSCertFile, s.TLSPrivateKeyFile)
			}
		}

		go func() {
			defer util.HandleCrash()
			for {
				// err == systemd.SdNotifyNoSocket when not running on a systemd system
				if err := systemd.SdNotify("READY=1\n"); err != nil && err != systemd.SdNotifyNoSocket {
					glog.Errorf("Unable to send systemd daemon successful start message: %v\n", err)
				}
				if err := secureServer.ListenAndServeTLS(s.TLSCertFile, s.TLSPrivateKeyFile); err != nil {
					glog.Errorf("Unable to listen for secure (%v); will try again.", err)
				}
				time.Sleep(15 * time.Second)
			}
		}()
	}
	handler := hapiserver.TimeoutHandler(m.InsecureHandler, longRunningTimeout)
	http := &http.Server{
		Addr:           insecureLocation,
		Handler:        hapiserver.RecoverPanics(handler),
		MaxHeaderBytes: 1 << 20,
	}
	if secureLocation == "" {
		// err == systemd.SdNotifyNoSocket when not running on a systemd system
		if err := systemd.SdNotify("READY=1\n"); err != nil && err != systemd.SdNotifyNoSocket {
			glog.Errorf("Unable to send systemd daemon successful start message: %v\n", err)
		}
	}
	glog.Infof("Serving insecurely on %s", insecureLocation)
	glog.Fatal(http.ListenAndServe())
	return nil
}
