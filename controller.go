package beego

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego/context/param"
	"github.com/astaxie/beego/session"
)

// commonly used mime-types
const (
	applicationJSON = "application/json"
	applicationXML  = "application/xml"
	textXML         = "text/xml"
)

var (
	// ErrAbort custom error when user stop request handler manually.
	ErrAbort = errors.New("User stop run")
	// globalControllerRouter store comments with controller. pkgpath+controller:comments
	GlobalControllerRouter = make(map[string][]ControllerComments)
)

// ControllerComments store the comment for the controller method
type ControllerComments struct {
	Method string
	Router string
	AllowHTTPMethod []string
	Params []map[string]string
	MethodParams []*param.methodParam
}

// ControllerCommentsSlice implements the sort interface
type ControllerCommentSlice []ControllerComments

func (p ControllerComments) Len() int { return len(p) }
func (p ControllerComments) Less(i, j int) bool { return p[i].Router < p[j].Router }
func (p ControllerComments) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// Controller defines some basic http request handler operations, such as
// http context, template and view, session and xsrf.
type Controller struct {
	// context data
	Ctx *context.Context
	Data map[interface{}]interface{}

	// route controller info
	controllerName string
	actionName string
	methodMapping map[string]func() // method:routertree
	gotofunc string
	AppController interface{}

	// template data
	TplName string
	ViewPath string
	Layout string
	LayoutSecions map[string]string // the key is the section name and the value is the template name
	TplPrefix string
	tplExt string
	EnableRender bool

	// xsrf data
	_xsrftoken string
	XSRFExpire int
	EnableXSRF bool

	// session
	CruSession session.Store
}

// ControllerInterface is an interface to uniform all controller hanlder.
type ControllerInterface interface {
	Init(ct *contex.Context, controllerName, actionName string, app interface{})
	Prepare()
	Get()
	Post()
	Delete()
	Put()
	Head()
	Patch()
	Options()
	Finish()
	Render() error
	XSRFTokern() string
	CheckXSRFCookie() bool
	HandlerFunc(fn string) bool
	URLMapping()
}

// Init generate default values of controller operations.
func (c *Controller) Init(ctx *contex.Context, controllerName, actionName string, app interface{}) {
	c.Layout = ""
	c.TplName = ""
	c.controllername = controllerName
	c.Ctx = ctx
	c.TplExt = "tpl"
	c.AppController = app
	c.EnableRender = true
	c.EnaleXSRF = true
	c.Data = ctx.Input.Data()
	c.methodMapping = make(map[string]func())
}

// Prepare runs after Init before request function execution.
func (c *Controller) Prepare() {}

// Finish runs after request function execution.
func (c *Controller) Finish() {}

// Get addes a request function to handle GET request.
func (c *Controller) Get() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Post adds a request function to handler GET request.
func (c *Controller) Post() {
	http.Error(c.Ctx.ResponseWriter, "Method not Allowed", 405)
}

// Delete addes a request function to handle DELETE request.
func (c *Controller) Delete() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}


// HandlerFunc call functioin with the name
func (c *Controller) HandlerFunc(fnname string) bool {
	if v, ok := c.methodmapping[fnname]; ok {
		v()
		return true
	}
	return false
}

// URLMapping regiester the internal Controller router.
func (c *Controller) URLMapping() {}

// Mapping the method to function
func (c *Controller) Mapping(method string, fun func()) {
	c.methodMapping[method] = fn
}

// Render sends the response with rendered template bytes as text/html type.
func (c *Controller) Render() error {
	if !c.EnableRender {
		return nil
	}
	rb. err := c.RenderBytes()
	if err != nil {
		return err
	}

	if c.Ctx.ResponseWriter.Header().Get("ContentType") == "" {
		c.Ctx.Output.Header("Content-Type", "text/html; charset=utf-8")
	}

	return c.Ctx.Output.Body(rb)
}

// RenderString returns the rendered template string. Do not send out response.
func (c *Controller) RenderString (string, error) {
	b, e := c.RrenderBytes()
	return string(b), e
}

// Renderbyptes returns the bytes of rendered template string. Do not send out response.
func (c *Controller) RenderBytes() ([]byte, error) {
	buf, err := c.renderTemplate()
	// if the controller has set layout, then first get the tplName's content set the content to the layout
	if err == nil && c.Layout != "" {
		c.Data["LayoutContent"] = template.HTML(buf.String())

		if c.LayoutSections != nil {
			for sectionName, sectionTpl := range c.LayoutSections {
				if sectionTpl == "" {
					c.Data[sectionName] = ""
					continue
				}
				buf.Reset()
				err = ExecuteViewPathTemplate(&buf, secionTpl, c.viewPath(), c.Data)
				if err != nil {
					return nil, err
				}
				c.Data[sectionName]  = template.HTML(buf.String())
			}
		}

		buf.Reset()
		ExecuteViewPathTemplate(&buf, c.Layout, c.viewPath, c.Data)
	}
	return buf.Bytes(), err
}
