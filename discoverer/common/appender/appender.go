package appender

import (
	dcommon "github.com/ebay/collectbeat/discoverer/common"

	"github.com/elastic/beats/libbeat/common"
)

// Appender take a module config and appends additional config parameters
type Appender interface {
	Append(config *dcommon.ConfigHolder)
}

type AppenderConstructor func(config *common.Config) (Appender, error)
