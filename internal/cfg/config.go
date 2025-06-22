package cfg

import "flag"

var TestFlag bool

func init() {
	flag.BoolVar(&TestFlag,
		"test",
		false,
		"Use to run app with separate user config file, useful for testing purposes",
	)
	flag.Parse()
}
