// getPara project getPara.go
package getPara

import (
	"flag"
	"os"
	"strconv"

	"simu/util/levlog"
)

func GetIntWithDefault(pval *int, name string, defaultVal int, usage string) {
	v := os.Getenv(name)
	if v == "" {
		flag.IntVar(pval, name, defaultVal, usage)
		return
	}
	val, err := strconv.Atoi(v)
	if err != nil {
		levlog.Error(err)
		val = defaultVal
	}
	flag.IntVar(pval, name, val, usage)
}

func GetStringWithDefault(pval *string, name, defaultVal string, usage string) {
	val := os.Getenv(name)
	if val == "" {
		val = defaultVal
	}
	flag.StringVar(pval, name, val, usage)
}

func GetFloatWithDefault(pval *float64, name string, defaultVal float64, usage string) {
	var val float64
	v := os.Getenv(name)
	if v == "" {
		flag.Float64Var(pval, name, defaultVal, usage)
		return
	}
	val, err := strconv.ParseFloat(v, 64)
	if err != nil {
		levlog.Error(err)
		val = defaultVal
	}
	flag.Float64Var(pval, name, val, usage)
}

func Finish() {
	flag.Parse()
}
