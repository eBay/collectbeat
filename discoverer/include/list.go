package include

import (
	// Include all builders
	_ "github.com/ebay/collectbeat/discoverer/kubernetes/common/builder/graphite_annotations"
	_ "github.com/ebay/collectbeat/discoverer/kubernetes/common/builder/log_annotations"
	_ "github.com/ebay/collectbeat/discoverer/kubernetes/common/builder/metrics_annotations"
	_ "github.com/ebay/collectbeat/discoverer/kubernetes/common/builder/metrics_secret"

	// Include all appenders
	_ "github.com/ebay/collectbeat/discoverer/kubernetes/common/appender/auth"
	_ "github.com/ebay/collectbeat/discoverer/kubernetes/common/appender/log_path"

	// Include all factories
	_ "github.com/ebay/collectbeat/discoverer/common/factory/cfgfile"
	_ "github.com/ebay/collectbeat/discoverer/common/factory/runner"
)
