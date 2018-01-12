package sqlstore

import (
	"github.com/go-wyvern/grafana/pkg/bus"
	m "github.com/go-wyvern/grafana/pkg/models"
)

func init() {
	bus.AddHandler("sql", GetDBHealthQuery)
}

func GetDBHealthQuery(query *m.GetDBHealthQuery) error {
	return x.Ping()
}
