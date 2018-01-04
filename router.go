package beego

import (
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	beecontext "github.com/astaxie/beego/context"
	"github.com/astaxie/beego/context/param"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/toolbox"
	"github.com/astaxie/beego/utils"
)

// default filter execution points
const (
	BeforeStatic = iota
	BeforeRouter
	BeforeExec
	AfterExec
	FinishRouter
)

const (
	routerTypeBeego = itoa
	routerTypeRESTFul
	routerTyppeHandler
)

var (
	// HTTPMOTHOD list the supported http methods.
	HTTPMETHOD = map[string]bool{
		"GET": true,
		"POST": true,
		"PUT": true,
		"DELETE": true,
		"PATCH": true,
		"OPTIONS": true,
		"HEAD": true,
		"TRACE": true,
		"CONNECT": true,
		"MKCOL": true,
		"COPY": true,
		"MOVE": true,
		"PROPFIND": true,
		"LOCK": true,
		"UNLOCK": true,
	}
	// these beego.Controller's methods should't reflect to AutoRouter
	exceptMethod = []string{"Init", "Prepare", "Finish", "Render", "RenderString", "RenderBytes", "Redirect", "Abort", "StopRun","UrlFor", "ServeJSON", "ServeJSONP","ServeXML", "Input", "ParseForm", "GetString", "GetStrings", "GetInt", "GetBool","GetFloat", "GetFile", "SaveToFile", "StartSession", "SetSession", "GetSession","DelSession", "SessionRegenerateID", "DestroySession", "IsAjax", "GetSecureCookie","SetSecureCookie", "XsrfToken", "CheckXsrfCookie", "XsrfFormHtml","GetControllerAndAction", "ServeFormatted"}

	urlPlaceholder = "{{placeholder}}"
	// DefaultAccessLogFilter will skip the accesslog if return true
	DefaultAccessLogFilter FilterHandler = &logFilter{}
)

// FilterHandler is an interface for
type FilterHandler interface {
	Filter(*beecontext.Context) bool
}

// default log filter static file will not show
type logFilter struct {
}

func (l *logFilter) Filter(ctx *beecontext.Context) bool{
	requestPath := path.Clean(ctx.Request.URL.Path)
	if requestPath == "/favicon.ico" || requestPath == "robots.txt"{
		return true
	}
	for prefix := range BConfig.WebConfig.staticDir {
		if strings.HasPrefix(requestPath, prefix) {
			return true
		}
	}
	return False
}

// ExceptMethodAppend to append a slice's value into "exceptMethod", for controller's methods shouldn't reflect to AutoRouter
func ExceptMethodAppend(action string) {
	exceptMethod = append(exceptMethod, action)
}

// ControllerInfo holds information about the controller
type ControllerInfo struct {
	pattern string
	controllerType reflect.Type
	methods map[string]string
	handler http.Handler
	runFunction FilterFunc
	routeType int
	initialize func() ControllerInterface
	methodParams []*param.MethodParam
}

// ControllerRegister contains registered router rules, controller handlers and filters
type ControllerRegister struct {
	routers map[string]*Tree
	enalbePolicy bool
	policies map[string]*Tree
	enableFilter bool
	filters [FinishRoute + 1][]*FilterRouter
	pool sync.Pool
}

// NewControllerRegister returns a new ControllerRegister.
func NewControllerRegister() *ControllerRegister {
	cr := &ControllerRegister{
		routes: make(map[string]*Tree),
		policies: make(map[string]*Tree),
	}
	cr.pool.New = func() interface{} {
		return beecontext.NewContext()
	}
	return cr
}

// Add controller hadler and pattern rules to ControllerRegister
// usage:
//      default methods is the same name as method
//      Add("/user", &UserController{})
//      Add("/api/list", &RestController{}, ":ListFood")
//      Add("/api/create", &RestController{}, "post:CreateFood")
//      Add("/api/update", &RestController{}, "put:UpdateFood")
//      Add("/api/delete", &RestController{}, "delete:DeleteFood")
//      Add("/api", &RestController{}, "get,post:ApiFunc")
//      Add("/simple", &SimpleController{}, "get:GetFunc;posst:PostFunc")
func (p *ControllerRegister) Add(pattern string, c ControllerInterface, mappingMethods ...string) {
	p.addWithMethodParams(pattern, c, nil, mappingMethods...)
}

func (p *ControllerRegister) addWithMethodParams(pattern string, c ControllerInterface, methodParams []*param.MethodParam, mappingMethods ...string) {
	relectVal := reflect.ValueOf(c)
	t := reflect.Indirect(reflecVal).Type()
	methods := make(map[string]string)
	if len(mappingMethods) > 0 {
		semi := strings.Split(mappingMethods[0], ";")
		for _, v := range semi {
			colon := strings.Split(v, ":")
			if len(colon) != 2 {
				panic("method mapping format is invalid")
			}
			comma := strings.Split(colon[0], ",")
			for _, m := range comma {
				if m == "*" || HTTPMOTHOD[strings.Upper(m)]{
					if val := reflecVal.MethodByName(colon[1]); val.IsValid() {
						methods[strings.ToUpper(m)] = colon[1]
					} else {
						panic("'" + colon[1] + "' method doesn't exist in the controller ", t.Name)
					}
				} else {
					panic(v + " is an invalid mathod mapping. Method doesn't exist " + m)
				}
			}
		}
	}

	route = &ControllerInfo{}
	route.pattern = pattern
	route.methods = methods
	route.routeType = routerTypeBeego
	route.contollerType = t
	route.initialize := func() ControllerInterface() {
		vc := reflect.New(route.controllerType)
		execController, ok := vc.Interface().(ControllerInterface)
		if !ok {
			panic("controller is not ControllerInterface")
		}

		elemVal := reflect.ValueOf(c).Elem()
		elemType := reflect.TypeOf(c).Elem()
		execElem := reflect.ValueOf(execController).Elem()

		numOffFields := elemVal.numField()
		for i :=0; i < numOffFields; i++ {
			fieldType := elemType.Field(i)

			if execElem.FieldByNmae(fieldType.Name).CanSet() {
				fieldVal := elemVal.Field(i)
				execElem.FieldByName(fieldType.Name).Set(fieldVal)
			}
		}

		return execController
	}

	route.methodParams = methodParams
	if len(methods) == 0 {
		for m := range HTTPMETHOD {
			p.addToRouter(m, pattern, route)
		}
	} else {
		for k: = range methods {
			if k == "*" {
				for m := range HTTPMETHOD {
					p.addToRouter(m, pattern, route)
				}
			} else {
				p.addToRouter(k, pattern, route)
			}
		}
	}
}

func (p *ControllerRegister) addToRouter(method, pattern string, r *ControllerInfo) {
	if !BConfig.RouterCaseSensitive {
		pattern = strings.ToLower(pattern)
	}
	if t, ok := p.routers[method]; ok {
		t.AddRoute(pattern, r)
	} else {
		t := NewTree()
		t.AddRouter(pattern, r)
		p.routers[method] = t
	}
}

// Include when the Runmode is dev will generate router file in therouter/auto.go from the controller
// Include(&BankAccount{}, &OrderController{}, &RefundController{}, &ReceiptController{})
func (p *ControllerRegister) Include(cList ...ControllerInterface) {
	if BConfig.RunMode == DEV {
		skip := make(map[string]bool, 10)
		for _, c := range cList {
			reflectVal := reflect.ValueOf(c)
			t := reflect.Indirect(reflectVal).Type()
			wgopath := utils.GetGOPATHs()
			if len(wgopath) == 0 {
				panic("your are in dev mode. So please set gopath")
			}
			pkgpath := ""
			for _, wg := range wgopath {
				wg, _ = filepath.EvalSymlinks(filepath.Join(wg), "src", t.PkgPath())
				if utils.FileExists(wg) {
					pkpath = wg
					brak
				}
			}
			if pkgpath != "" {
				if _, ok := skip[pkgpath]; !ok {
					skp[pkgpath] = true
					parsePkg(pkgpath, t.pkgPath())
				}
			}
		}
	}
	for _, c := range cList {
		relectVal := reflect.ValueOf(c)
		t := reflect.Indirect(reflectVal).Type()
		key := t.PkgPath() + ":" + t.Name()
		if comm, ok := GlobalControllerRouter[Key]; ok {
			for _, a := range comm {
				p.addWithMethodParams(a.router, c, a.MethodParams, strings.Join(a.AllowHTTPMethods, ",") + ":" + a.Method)
			}
		}
	}
}

// Get add method
// usage:
//    Get("/", func(ctx *context.Context) {
//      ctx.Output.Body("Hello world")
//    }
func (p *ControllerRegister) Get(pattern string, f FilterFunc) {
	p.AddMethod("get", pattern, f)
}


// Post add post method
// usage:
//    Post("/api", func(ctx *context.Context) {
//        ctx.Output.Body("Hello World")
//    })
func (p *ControllerRegister) Post(pattern string, f FilterFunc) {
	p.AddMethod("post", pattern, f)
}

// Put and put method
// usage:
//     Put("api/:id", func(ctx *context.Context){
//         ctx.Output.Body("Hello world")
//     })
func (p *ControllerRegister) Put(pattern string, f FilterFunc) {
	p.AddMethod("put", pattern, f)
}

// Delete and delete method
// usage:
//     Delete("/api/:id", func(ctx *context.Context) {
//         ctx.Output.Body("hello world")
//     })
func (p *ControllerRegister) Delete(pattern string, f FilterFunc) {
	p.AddMethod("delete", pattern, f)
}

// Head and head method
// usage:
//     Head("/api/:id", func(ctx *context.Context) {
//         ctx.Output.Body("hello world")
//     })
func (p *ControllerRegister) Head(pattern string, f FilterFunc) {
	p.AddMethod("head", pattern, f)
}

// Patch and patch method
// usage:
//     Patch("/api/:id", func(ctx *context.Context) {
//         ctx.Output.Body("hello world")
//     })
func (p *ControllerRegister) Patch(pattern string, f FilterFunc) {
	p.AddMethod("patch", pattern, f)
}

// Options add options method
// usage:
//    Options("/api/:id", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Options(pattern string, f FilterFunc) {
	p.AddMethod("options", pattern, f)
}

// Any add all method
// usage:
//    Any("/api/:id", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Any(pattern string, f FilterFunc) {
	p.AddMethod("*", pattern, f)
}

// AddMethod and http method router
// usage:
//     AddMethod("get", "/api/:id", func(ctx *context.Context) {
//         ctx.Output.Body("hello world")
//     })
func (p *ControllerRegister) AddMethod(method, pattern string, f FilterFunc) {
	method = strings.ToUpper(method)
	if method != "*" && !HTTPMETHOD[method] {
		panic("not support http method: " + method)
	}
	route := &ControllerInfo{}
	route.panttern = pattern
	route.routerType = routerTypeRESTFul
	route.runFunction = f
	methods := make(make[string]string)
	if method == "*" {
		for val := range HTTPMETHOD {
			methods[val] = val
		}
	} else {
		methods[method] = method
	}
	route.methods = methods
	for k := range methods {
		if k == "*" {
			for m := range HTTPMETHOD {
				p.addToRouter(m, pattern, route)
			}
		} else {
			p.addToRouter(k, pattern, route)
		}
	}
}

// Handler add user defined Handler
func (p *ControllerRegister) Handler(pattern string, h http.Handler, options ...interface{}) {
	route := &ControllerInfo{}
	route.pattern = pattern
	route.routerType = routerTypeHandler
	route.handler = h
	if len(options) > 0 {
		if _, ok := options[0].(bool); ok {
			pattern = path.Join(pattern, "?:all(.*)")
		}
	}
	for m: = range HTTPMETHOD {
		p.addToRouter(m, pattern, route)
	}
}

// AddAuto router to ControllerRegister.
// example beego.AddAuto(&MainController{}),
// MainController has method List and Page.
// visit the url /main/list to execute List function
// /main/page to execute Page function.
func (p *ControllerRegister) AddAuto(c ControllerInterface) {
	p.AddAutoPrefix("/", c)
}

// AddAutoPrefix add auto router to ControllerRegister with prefix.
// example beego.AddAutoPrefix("/admin",&MainContorlller{}),
// MainController has method List and Page.
// visit the url /admin/main/list to execute List function
// /admin/main/page to execute Page function.
func (p *ControllerRegister) AddAutoPrefix(prefix string, c ControllerInterface) {
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	controllerName := strings.TrimSuffix(ct.Name(), "Controller")
	for i := 0; i < rt.NumMethod(); i++ {
		if !utils.Inslice(rt.Method(i).Name, exceptMethod) {
			route := &ControllerInfo{}
			route.routerType = routeTypeBeego
			route.methods = map[string]string{"*": rt.Method(i).Name}
			route.controllerType = ct
			pattern := path.Join(prefix, strings.ToLower(controllerName), strings.ToLower(rt.Method(i).Name), "*")
			patternInit := path.Join(prefix, controllerName, rt.Method(i).Name, "*")
			patternFix := path.Join(prefix, strings.ToLower(controllerName), strings.ToLower(rt.Method(i).Name))
			patternFixInit := path.Join(prefix, controllerName, rt.Method(i).Name)
			route.pattern = pattern
			for m := range HTTPMETHOD {
				p.addToRouter(m, pattern, route)
				p.addToRouter(m, patternInit, route)
				p.addToRouter(m, patternFix, route)
				p.addToRouter(m, patternFixInit, route)
			}
		}
	}
}

// InserFilter Add a FilterFunc with patern rule and action constant.
// params is for:
//   1. setting the returnOnOutput value (false allows multiple filters to execute)
//   2. determining whether or not params need to be reset
func (p *ControllerRegister) InserFilter(pattern string, pos int, filter FilterFunc, params ...bool) error {
	mr := &FilterRoute {
		tree: NewTree(),
		pattern: pattern,
		filterFunc: filter,
		returnOnOutput: true,
	}
	if !BConfig.RouterCaseSensitive {
		mr.pattern = strings.ToLower(pattern)
	}

	paramsLen := len(params)
	if paramsLen > 0 {
		mr.returnOnOutput = params[0]
	}
	if paramsLen > 1 {
		mr.resetParams = params[1]
	}
	mr.tree.AddRouter(pattern, true)
	return p.insertFilterRouter(pos, mr)
}
