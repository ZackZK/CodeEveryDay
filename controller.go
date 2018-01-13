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
