package releases

import (
	"io/ioutil"
	"log"
	"time"
)

var (
	defaultInstallTimeout = 30 * time.Second
	defaultListTimeout    = 10 * time.Second
	discardLogger         = log.New(ioutil.Discard, "", 0)
)
