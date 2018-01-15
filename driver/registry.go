package driver

import (
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"sync"
)

var driversMu sync.Mutex
var drivers = make(map[string]*drv)
var driversPkg = make(map[string]*drv) // maps driver's package path to driver

// FactoryFunc produces driver instances
type FactoryFunc func(url string) (Driver, error)

type drv struct {
	new           FactoryFunc
	fileExtension string
	fileTemplate  []byte

	// pkg is a package path of driver
	pkg string
}

// Register a driver so it can be created from its name. Drivers should call
// this from an init() function so that they registers themselves on import.
// filenameExtension returns the extension of the migration files.
// migrationFileTemplate is a content that should be written
// into newly-created migration file (can be nil).
func Register(driverName, migrationFileExtension string, migrationFileTemplate []byte, f FactoryFunc) {
	migrationFileExtension = normalizeFilenameExtension(migrationFileExtension, driverName)
	driversMu.Lock()
	defer driversMu.Unlock()
	if f == nil {
		panic("driver: Tried to register nil driver factory " + driverName)
	}
	if _, dup := drivers[driverName]; dup {
		panic("driver: Register called twice for f " + driverName)
	}
	ndrv := &drv{
		new:           f,
		fileExtension: migrationFileExtension,
		fileTemplate:  migrationFileTemplate,
		pkg:           callerPkg(driverName), // we need this in order to find driver by instance
	}
	drivers[driverName] = ndrv
	driversPkg[ndrv.pkg] = ndrv
}

// getDriver retrieves a registered driver by name.
func getDriver(name string) *drv {
	driversMu.Lock()
	defer driversMu.Unlock()
	driver := drivers[name]
	return driver
}

// registeredDrivers returns a sorted list of the names of the registered drivers.
func registeredDrivers() []string {
	driversMu.Lock()
	defer driversMu.Unlock()
	var list []string
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

// normalizeFilenameExtension panics if the driver's filename extension
// is not correct or empty.
func normalizeFilenameExtension(ext, driverName string) string {
	if ext[0:1] == "." {
		ext = string(ext[1:])
	}
	if ext == "" {
		panic(fmt.Sprintf("%s migrationFileExtension is empty string", driverName))
	}
	return ext
}

var callerPkgRegex = regexp.MustCompile(`^(.*)\.\w+\.\d+$`)

func callerPkg(driverName string) string {
	fpcs := make([]uintptr, 1)
	n := runtime.Callers(3, fpcs)
	if n == 0 {
		panic("failed to determine package of driver " + driverName)
	}

	fun := runtime.FuncForPC(fpcs[0] - 1)
	if fun == nil {
		panic("failed to determine package of driver " + driverName)
	}
	sbm := callerPkgRegex.FindStringSubmatch(fun.Name())
	if len(sbm) < 2 {
		panic("failed to determine package of driver " + driverName)
	}
	return sbm[1]
}
