package metagen

import "github.com/elastic/beats/libbeat/common"

type MetaGen interface {
	GetMetaData(string) common.MapStr
}
