package promethuesexpoter

import (
	"net/http"

	"github.com/chentao-kernel/spycat/pkg/component/exporter"
	"github.com/chentao-kernel/spycat/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
)

type PromethuesExporter struct {
	cfg      *Config
	labels   []attribute.KeyValue
	registry *prometheus.Registry
	handler  http.Handler
	stopFunc func() error
}

func NewExporter(config interface{}) exporter.Exporter {
	cfg, ok := config.(*Config)
	if !ok {
		log.Loger.Error("Convert %v config failed", cfg)
		return nil
	}
}
