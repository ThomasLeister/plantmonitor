/*
 * Common testing init routines
 */
package testing_init

import (
	"fmt"
	"os"
	"path"
	"runtime"
)

/*
 * init() function is run whenever this package has been included in another package.
 * It is solely used in _test.go files of packages and will automatically chdir() to the main directory "plantmonitor" for execution.
 * This enables us to use the same hardcoded directory paths e.g. for config.yaml as in the non-testing parts of the application (main.go).
 */
func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename) + "/../")
	fmt.Println("test_init: chdir() to: ", dir)
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}
