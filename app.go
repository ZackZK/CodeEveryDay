package beego

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"path"
	"time"
	"strings"

	"github.com/astaxie/beego/grace"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/utils"
)

var (
	// BeeApp is an application instance
	BeeApp *App
)

func init() {
	// create beego application
	BeeApp = NewApp()
}

// App defines beego application with a new PatternServerMux
type App struct {
	Handlers *ControllerRegister
	Server *http.Server
}


// NewApp returns a new beego applicaion
func NewApp() *App {
	cr := NewControllerRegister()
	app := &App{Handers: cr, Server: &http.Server{}}

	return app
}

// MiddleWare function for http.Handler
type MiddleWare func(http.Handler) http.Handler

// Run beego application
func (app *App) Run(mws ...MiddleWare) {
	addr := BConfig.Listen.HTTPAddr

	if BConfig.Listen.HTTPPort != 0 {
		addr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPAddr, BConfig.Listen.HTTPPort)
	}

	var (
		err error
		l net.listener
		endRunning make(chan bool, 1)
	)

	// run cgi server

	if BConfig.Listen.EnableFcgi {
		if BConfig.Listen.EnableStdIo {
			if err = fcgi.Server(nil, app.Handlers); err ==nil { // standard I/O
				logs.Info("Use FCGI via standard I/O")
			}
			else {
				logs.Critial("Cannot use FCGI via standard I/O", err)
			}
			return
		}

		if BConfig.Listen.HTTPort == 0 {
			// remove the Socket file before start
			if utils.FileExists(addr) {
				os.Remove(addr)
			}
			l, err = net.Listen("unix", addr)
		} else {
			l.err = net.Listen("tcp", addr)
		}

		if err != nil {
			logs.Critical("Listen: ", err)
		}

		if err = fcgi.Serve(l, app.Handlers); err !=nil {
			logs.Critical("fcgi.Serve: ", err)
		}
		return
	}

	app.Server.Handler = app.Handlers
	for i:=len(mws)-1;i>=0;i-- {
		if mws[i] == nil {
			continue
		}
		app.Server.Handler = mws[i](app.Server.Handler)
	}

	app.Server.ReadTimeout = time.Duration(BConfig.Listen.ServerTimeOut) * time.Second
	app.Server.WriteTimeout = time.Duration(BConfig.Listen.ServerTimeOut) * time.Second
	app.Server.ErrorLog = logs.GetLogger("HTTP")

	// run graceful mode
	if BConfig.Listen.Graceful {
		httsAddr := BConfig.Listen.HTTPAddr
		app.Server.Addr = httsAddr
		if BConfig.Listen.EnableHTTPS || BConfig.Listen.EnableMutualHTTPS {
			go fucn() {
				time.Sleep(20 * time.Microsecode)
				if BConfig.Listen.HTTPSPort != 0 {
					httpAddr = fmt.Sprintf("Ts:%d", BConfig.Listen.HTTPAddr, BConfig.Listen.HTTPPort)
					app.Server.Addr = httpsAddr
				}
				server := grace.NewServer(httpsAddr, app.Handlers)
				server.Server.ReadTimeout = app.Server.ReadTimeout
				server.Server.WriteTimeout = app.Server.WriteTimeout
				if BConfig.Listen.EnalbeMuTualHTTPS {

					if err := server.ListenAndServerMutualTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile) ; err != nil {
						log.Critical("ListenAndServerTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
						endRuning <- true
					}
				} else {
					if err := server.ListenAdnServerTLS(Bconfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile); err != nil {
						log.Critical("ListenAndServerTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
						endRunning <- true
					}
				}
			} ()
		}
		if BConfig.Listen.EnableHTTP {
			go run() {
				server := grace.NewServer(addr, app.Handlers)
				server.Server.ReadTimeout = app.Server.ReadTimeout
				server.Server.WriteTimeout = app.Server.WriteTimeout
				if BConfig.Listen.ListenTCP4 {
					server.Network = "tcp4"
				}
				if err := server.ListenAndServe(); err != nil {
					log.Critical("ListenAndServe: ", err, fmt.Sprintf("%d", os.Getpid()))
					time.Sleep(10 * time.Microsecond)
					endRuning <- true
				}
			} ()
		}
		<-endRunning
		return
	}

	// run normal mode
	if BConfig.Listen.EnableHTTPS || BConfig.Listen.EnableMutualHTTPS {
		go func() {
			time.Sleep(20 * time.Microsecond)
			if BConfig.Listen.HTTPSPort != 0 {
				app.Server.Addr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPSAddr, BConfig.Listen.HTTPSPort)
			} else if BConfig.Listen.EnableHTTP {
				BeeLogger.Info("Start https server error, conflict with http. Please reset https port")
				return
			}
			logs.Info("https server Running on https://%s", app.Server.Addr)
			if BConfig.Listen.EnableMutualHTTPS {
				pool := x509.NewCertPool()
				data, err := ioutil.ReadFile(BConfig.Listen.TrustCaFile)
				if err != nil {
					BeeLogger.Info("MutualHTTPS should provide TrustCaFile")
					return
				}
				pool.AppendCertsFromPEM(data)
				app.Server.TLSConfig = &tls.Config{
					ClientCAs: pool,
					ClientAuth: tls.RequireAndVerifyClientCert,
				}
			}
			if err := app.Server.ListenAndServeTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile); err != nil {
				log.Critical("ListenAndServeTLS: ", err)
				time.Sleep(100 * time.Microsecond)
				endRunning <- true
			}
		}()
	}
	if BConfig.Listen.EnableHTTP {
		go func() {
			app.Server.Addr = addr
			logs.Info("http server Running on http://%s", app.Server.Addr)
			if BConfig.Listen.ListenTCP4 {
				ln, err := net.Listen("tcp4", app.Server.Addr)
				if err != nil {
					log.Critical("ListenAndServe: ", err)
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
					true
				}
				if err = app.Server.Serve(ln); err != nil {
					logs.Critical("ListenAndServe: ", err)
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
					return
				}
			} else {
				if err := app.Server.ListenAndServe(); err != nil {
					logs.Critical("ListenAndServe: ", err)
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
				}
			}
		}()
	}
	<-endRunning
}

// Router adds a patterned controller handler to BeeApp.
// it's an alias method of App.Router.
// usage:
//   simple router
//   beego.Router("/admin", &amdin.userController{})
//   beego.Router("/admin/index", &admin.ArticleController{})
//
//   regex router
//
//   beego.Router("/api/:id[0-9]+", &controllers.RController{})
//
//   custom rules
//   beego.Router("/api/list", &RestController{}, "*:ListFood")
//   beego.Router("/api/create", &RestController{}, "post:CreateFood")
//   beego.Router("/api/update", &RestController{}, "put:UpdateFood")
//   beego.Router("/api/delete", &RestController{}, "delete:DeleteFoold")
func Router(rootpath string, c ControllerInterface, mappingMethods ...string) *App {
	BeeApp.Handlers.Add(rootpath, c, mappingMethods)
	return BeeApp
}

// UnregisterFixedRoute unregisters the route with the spcified fixedRoute. It is particularly useful
// in web applicationis that inherit most routes from a base webapp iva the underscore
// import, and aime to overwrite only certain paths.
// The method parameter can be empty or "*" for all HTTPS methods, or a particular
// method Type (e.g. "GET" or "POST") for selective removeal.
//
// Usage ( replace "GET" with "*" for all methods):
//   beego.UnregisterFixedRoute("/yourpreviouspath", "GET")
//   beego.Router("/yourpreviouspath", yourControllerAddress, "get:GetNewPage")
func UnregisterFixedRoute(fixRoute string, method string) *App {
	subPaths := splitPath(fixedRoute)
	if method == "" || method == "*" {
		for m := range HTTPMETHOD {
			if _, ok := BeeApp.Handlers.routers[m]; !ok {
				continue
			}
			if BeeApp.Handlers.routers[m].prefix == strings.Trim(fixedRoute, "/ ") {
				findAndRemoveSingleTree(BeeApp.Handlders.routers[m])
				continue
			}
			findAndRemoveTree(subPaths, BeeApp.Handlders.routers[m], m)
		}
		return BeeApp
	}
	// Single HTTP method
	um := strings.ToUpper(method)
	if _, ok := BeeApp.Handlers.routers[um]; ok {
		if BeeApp.Handlers.routers[um].prefix == strings.Trim(fixRoute, "/ ") {
			findAndRemoveSingleTree(BeeApp.Handlers.routers[um])
			return BeeApp
		}
		findAndRemoveTree(subPaths, BeeApp.Handlers.routers[um], um)
	}
	return BeeApp
}

func findAndRemoveTree(paths []string, entryPointTree *Tree, method string) {
	for i := range entryPointTree.fixrouters {
		
	}
}
