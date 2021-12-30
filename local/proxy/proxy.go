package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/elazarl/goproxy"
	"github.com/pkg/errors"
	"github.com/symfony-cli/cert"
	"github.com/symfony-cli/symfony-cli/local/html"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/local/projects"
)

type Proxy struct {
	*Config
	proxy *goproxy.ProxyHttpServer
}

func tlsToLocalWebServer(proxy *goproxy.ProxyHttpServer, tlsConfig *tls.Config, localPort int) *goproxy.ConnectAction {
	httpError := func(w io.WriteCloser, ctx *goproxy.ProxyCtx, err error) {
		if _, err := io.WriteString(w, "HTTP/1.1 502 Bad Gateway\r\n\r\n"); err != nil {
			ctx.Warnf("Error responding to client: %s", err)
		}
		if err := w.Close(); err != nil {
			ctx.Warnf("Error closing client connection: %s", err)
		}
	}
	connectDial := func(proxy *goproxy.ProxyHttpServer, network, addr string) (c net.Conn, err error) {
		if proxy.ConnectDial != nil {
			return proxy.ConnectDial(network, addr)
		}
		if proxy.Tr.Dial != nil {
			return proxy.Tr.Dial(network, addr)
		}
		return net.Dial(network, addr)
	}
	// tlsRecordHeaderLooksLikeHTTP reports whether a TLS record header
	// looks like it might've been a misdirected plaintext HTTP request.
	tlsRecordHeaderLooksLikeHTTP := func(hdr [5]byte) bool {
		switch string(hdr[:]) {
		case "GET /", "HEAD ", "POST ", "PUT /", "OPTIO":
			return true
		}
		return false
	}
	return &goproxy.ConnectAction{
		Action: goproxy.ConnectHijack,
		Hijack: func(req *http.Request, proxyClient net.Conn, ctx *goproxy.ProxyCtx) {
			proxyClientTls := tls.Server(proxyClient, tlsConfig)
			if err := proxyClientTls.Handshake(); err != nil {
				defer proxyClient.Close()
				if re, ok := err.(tls.RecordHeaderError); ok && re.Conn != nil && tlsRecordHeaderLooksLikeHTTP(re.RecordHeader) {
					io.WriteString(proxyClient, "HTTP/1.0 400 Bad Request\r\n\r\nClient sent an HTTP request to an HTTPS server.\n")
					return
				}

				ctx.Logf("TLS handshake error from %s: %v", proxyClient.RemoteAddr(), err)
				return
			}

			ctx.Logf("Assuming CONNECT is TLS, TLS proxying it")
			targetSiteCon, err := connectDial(proxy, "tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
			if err != nil {
				targetSiteCon.Close()
				httpError(proxyClientTls, ctx, err)
				return
			}

			negotiatedProtocol := proxyClientTls.ConnectionState().NegotiatedProtocol
			if negotiatedProtocol == "" {
				negotiatedProtocol = "http/1.1"
			}

			targetTlsConfig := &tls.Config{
				RootCAs:    tlsConfig.RootCAs,
				ServerName: "localhost",
				NextProtos: []string{negotiatedProtocol},
			}

			targetSiteTls := tls.Client(targetSiteCon, targetTlsConfig)
			if err := targetSiteTls.Handshake(); err != nil {
				ctx.Warnf("Cannot handshake target %v %v", req.Host, err)
				httpError(proxyClientTls, ctx, err)
				targetSiteTls.Close()
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				if _, err := io.Copy(proxyClientTls, targetSiteTls); err != nil {
					ctx.Warnf("Error copying to target: %s", err)
					httpError(proxyClientTls, ctx, err)
				}
				proxyClientTls.CloseWrite()
				wg.Done()
			}()
			go func() {
				if _, err := io.Copy(targetSiteTls, proxyClientTls); err != nil {
					ctx.Warnf("Error copying to client: %s", err)
				}
				targetSiteTls.CloseWrite()
				wg.Done()
			}()
			wg.Wait()
			proxyClientTls.Close()
			targetSiteTls.Close()
		},
	}
}

func New(config *Config, ca *cert.CA, logger *log.Logger, debug bool) *Proxy {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = debug
	proxy.Logger = logger
	p := &Proxy{
		Config: config,
		proxy:  proxy,
	}

	var proxyTLSConfig *tls.Config

	if ca != nil {
		goproxy.GoproxyCa = *ca.AsTLS()
		getCertificate := p.newCertStore(ca).getCertificate
		cert, err := x509.ParseCertificate(ca.AsTLS().Certificate[0])
		if err != nil {
			panic(err)
		}
		certpool := x509.NewCertPool()
		certpool.AddCert(cert)
		tlsConfig := &tls.Config{
			RootCAs:        certpool,
			GetCertificate: getCertificate,
			NextProtos:     []string{"http/1.1", "http/1.0"},
		}
		proxyTLSConfig = &tls.Config{
			RootCAs:        certpool,
			GetCertificate: getCertificate,
			NextProtos:     []string{"h2", "http/1.1", "http/1.0"},
		}
		goproxy.MitmConnect.TLSConfig = func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
			return tlsConfig, nil
		}
		// They don't use TLSConfig but let's keep them in sync
		goproxy.OkConnect.TLSConfig = goproxy.MitmConnect.TLSConfig
		goproxy.RejectConnect.TLSConfig = goproxy.MitmConnect.TLSConfig
		goproxy.HTTPMitmConnect.TLSConfig = goproxy.MitmConnect.TLSConfig
	}
	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "" {
			fmt.Fprintln(w, "Cannot handle requests without a Host header, e.g. HTTP 1.0")
			return
		}
		r.URL.Scheme = "http"
		r.URL.Host = r.Host
		if r.URL.Path == "/proxy.pac" {
			p.servePacFile(w, r)
			return
		} else if r.URL.Path == "/" {
			p.serveIndex(w, r)
			return
		}
		http.Error(w, "Not Found", 404)
	})
	cond := proxy.OnRequest(config.tldMatches())
	cond.HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		hostName, hostPort, err := net.SplitHostPort(host)
		if err != nil {
			// probably because no port in the host (determine it via the scheme)
			if ctx.Req.URL.Scheme == "https" {
				hostPort = "443"
			} else {
				hostPort = "80"
			}
			hostName = ctx.Req.Host
		}
		// wrong port?
		if ctx.Req.URL.Scheme == "https" && hostPort != "443" {
			return goproxy.MitmConnect, host
		} else if ctx.Req.URL.Scheme == "http" && hostPort != "80" {
			return goproxy.MitmConnect, host
		}
		projectDir := p.GetDir(hostName)
		if projectDir == "" {
			return goproxy.MitmConnect, host
		}

		pid := pid.New(projectDir, nil)
		if !pid.IsRunning() {
			return goproxy.MitmConnect, host
		}

		backend := fmt.Sprintf("127.0.0.1:%d", pid.Port)

		if hostPort != "443" {
			// No TLS termination required, let's go trough regular proxy
			return goproxy.OkConnect, backend
		}

		if proxyTLSConfig != nil {
			return tlsToLocalWebServer(proxy, proxyTLSConfig, pid.Port), backend
		}

		// We didn't manage to get a tls.Config, we can't fulfill this request hijacking TLS
		return goproxy.RejectConnect, backend
	})
	cond.DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		hostName, hostPort, err := net.SplitHostPort(r.Host)
		if err != nil {
			// probably because no port in the host (determine it via the scheme)
			if r.URL.Scheme == "https" {
				hostPort = "443"
			} else {
				hostPort = "80"
			}
			hostName = r.Host
		}
		// wrong port?
		if r.URL.Scheme == "https" && hostPort != "443" {
			return r, goproxy.NewResponse(r,
				goproxy.ContentTypeHtml, http.StatusNotFound,
				html.WrapHTML(
					"Proxy Error",
					html.CreateErrorTerminal(`You must use port 443 for HTTPS requests (%s used)`, hostPort)+
						html.CreateAction(fmt.Sprintf("https://%s/", hostName), "Go to port 443"), ""),
			)
		} else if r.URL.Scheme == "http" && hostPort != "80" {
			return r, goproxy.NewResponse(r,
				goproxy.ContentTypeHtml, http.StatusNotFound,
				html.WrapHTML(
					"Proxy Error",
					html.CreateErrorTerminal(`You must use port 80 for HTTP requests (%s used)`, hostPort)+
						html.CreateAction(fmt.Sprintf("http://%s/", hostName), "Go to port 80"), ""),
			)
		}
		projectDir := p.GetDir(hostName)
		if projectDir == "" {
			hostNameWithoutTLD := strings.TrimSuffix(hostName, "."+p.TLD)
			hostNameWithoutTLD = strings.TrimPrefix(hostNameWithoutTLD, "www.")

			// the domain does not refer to any project
			return r, goproxy.NewResponse(r,
				goproxy.ContentTypeHtml, http.StatusNotFound,
				html.WrapHTML("Proxy Error", html.CreateErrorTerminal(`# The "%s" hostname is not linked to a directory yet.
# Link it via the following command:

<code>symfony proxy:domain:attach %s --dir=/some/dir</code>`, hostName, hostNameWithoutTLD), ""))
		}

		pid := pid.New(projectDir, nil)
		if !pid.IsRunning() {
			return r, goproxy.NewResponse(r,
				goproxy.ContentTypeHtml, http.StatusNotFound,
				// colors from http://ethanschoonover.com/solarized
				html.WrapHTML(
					"Proxy Error",
					html.CreateErrorTerminal(`# It looks like the web server associated with the "%s" hostname is not started yet.
# Start it via the following command:

$ symfony server:start --daemon --dir=%s`,
						hostName, projectDir)+
						html.CreateAction("", "Retry"), ""),
			)
		}

		r.URL.Host = fmt.Sprintf("127.0.0.1:%d", pid.Port)

		if r.Header.Get("X-Forwarded-Port") == "" {
			r.Header.Set("X-Forwarded-Port", hostPort)
		}

		return r, nil
	})
	return p
}

func (p *Proxy) Start() error {
	go p.Config.Watch()
	return errors.WithStack(http.ListenAndServe(":"+strconv.Itoa(p.Port), p.proxy))
}

func (p *Proxy) servePacFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/x-ns-proxy-autoconfig")
	w.Write([]byte(fmt.Sprintf(`// Only proxy *.%s requests
// Configuration file in ~/.symfony5/proxy.json
function FindProxyForURL (url, host) {
	if (dnsDomainIs(host, '.%s')) {
		return 'PROXY %s:%d';
	}
	return 'DIRECT';
}
`, p.TLD, p.TLD, p.Host, p.Port)))
}

func (p *Proxy) serveIndex(w http.ResponseWriter, r *http.Request) {
	content := ``

	proxyProjects, err := ToConfiguredProjects()
	if err != nil {
		return
	}
	runningProjects, err := pid.ToConfiguredProjects()
	if err != nil {
		return
	}
	projects, err := projects.GetConfiguredAndRunning(proxyProjects, runningProjects)
	if err != nil {
		return
	}
	projectDirs := []string{}
	for dir := range projects {
		projectDirs = append(projectDirs, dir)
	}
	sort.Strings(projectDirs)

	content += "<table><tr><th>Directory<th>Port<th>Domains"
	for _, dir := range projectDirs {
		project := projects[dir]
		content += fmt.Sprintf("<tr><td>%s", dir)
		if project.Port > 0 {
			content += fmt.Sprintf(`<td><a href="http://127.0.0.1:%d/">%d</a>`, project.Port, project.Port)
		} else {
			content += `<td style="color: #b58900">Not running`
		}
		content += "<td>"
		for _, domain := range project.Domains {
			if strings.Contains(domain, "*") {
				content += fmt.Sprintf(`%s://%s/`, project.Scheme, domain)
			} else {
				content += fmt.Sprintf(`<a href="%s://%s/">%s://%s/</a>`, project.Scheme, domain, project.Scheme, domain)
			}
			content += "<br>"
		}
	}
	w.Write([]byte(html.WrapHTML("Proxy Index", html.CreateTerminal(content), "")))
}
