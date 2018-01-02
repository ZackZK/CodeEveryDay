package beego

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	VERSION = "1.9.2"
	DEV = "dev"
	PROD = "PROD"
)

type hookfunc func() error

var (
	hooks = make([]hookfunc, 0)
)

// AddAPPStartHook is used to register the hookfunc
// The hookfunc will run in beego.Run()
// such as initiating sessin, starting middleware, building template, starting admin control and so on

func AddAPPStartHook(hf ...hookfunc) {
	hooks = append(hooks, hf...)
}

// Run Beego application
// beego.Run() default run on HttpPort
// beego.Run("localhost")
// beego.Run(":8089")
// beego.Run("127.0.0.1:8089")

func Run(params ...string) {

	initBeforeHTTPRun()

	if len(params) > 0 && params[0] != "" {
		strs := strings.Split(params[0], ":")

		if len(strs) > 0 && strs[0] != "" {
			BConfig.Listen.HTTPAddr = strs[0]
		}
		if len(strs) > 1 && strs[1] != "" {
			BConfig.Listen.HTTPPort, _ = strconv.Atoi(strs[1])
		}
	}

	BeeApp.Run()
}

func RunWithMiddleWares(addr string, mws ...MiddleWare) {
	initBeforeHTTPRun()

	strs := strings.Split(addr, ":")

	if len(strs) > 0 && strs[0] != "" {
		Bconfig.Listen.HTTPAddr = str[0]
	}
	if len(strs) > 1 && strs[1] != "" {
		BConfig.Listen.HTTPPort, _ = strconv.Atoi(strs[1])
	}

	BeeApp.Run(mws...)
}

func initBeforeHTTPRun() {
	// init hooks
	AddAPPStartHook(
		registerMine,
		registerDefaultErrorHandler,
		registerSession,
		registerTemplate,
		registerAdmin,
		registerGzip
	)

	for _, hk := range hooks {
		if err := hk(); err != nil {
			panic(err)
		}
	}
}

// TestBeegoInit is for test pacage init
func TestBeegoInit(ap string) {
	path := filepath.Join(ap, "conf", "app.conf")
	os.Chdir(ap)
	InitBeegoBeforeTest()
}

// InitBeegoBeforeTest is for test pacage init
func InitBeegoBeforeTest(appConfigPath string) {
	if err := LoadAppConfig(appConfigProvider, appConfigPath); err != nil {
		panic(err)
	}

	Beconfig.RunMode = "test"
	initBeforeHTTPRun()
}
