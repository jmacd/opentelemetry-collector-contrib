// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datasenders // import "github.com/open-telemetry/opentelemetry-collector-contrib/testbed/datasenders"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
<<<<<<< HEAD
=======
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.uber.org/zap"
>>>>>>> 08cd036e9da7fa0678402fe8ce1d4d7bd3234a6a

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/testbed"
)

// jaegerGRPCDataSender implements TraceDataSender for Jaeger thrift_http exporter.
type jaegerGRPCDataSender struct {
	testbed.DataSenderBase
	consumer.Traces
}

// Ensure jaegerGRPCDataSender implements TraceDataSender.
var _ testbed.TraceDataSender = (*jaegerGRPCDataSender)(nil)

// NewJaegerGRPCDataSender creates a new Jaeger exporter sender that will send
// to the specified port after Start is called.
func NewJaegerGRPCDataSender(host string, port int) testbed.TraceDataSender {
	return &jaegerGRPCDataSender{
		DataSenderBase: testbed.DataSenderBase{Port: port, Host: host},
	}
}

func (je *jaegerGRPCDataSender) Start() error {
	factory := jaegerexporter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*jaegerexporter.Config)
	// Disable retries, we should push data and if error just log it.
	cfg.RetrySettings.Enabled = false
	// Disable sending queue, we should push data from the caller goroutine.
	cfg.QueueSettings.Enabled = false
	cfg.Endpoint = je.GetEndpoint().String()
	cfg.TLSSetting = configtls.TLSClientSetting{
		Insecure: true,
	}
<<<<<<< HEAD
=======
	params := exportertest.NewNopCreateSettings()
	params.Logger = zap.L()
>>>>>>> 08cd036e9da7fa0678402fe8ce1d4d7bd3234a6a

	exp, err := factory.CreateTracesExporter(context.Background(), testbed.ExporterCreateSettings(), cfg)
	if err != nil {
		return err
	}

	je.Traces = exp
	return exp.Start(context.Background(), componenttest.NewNopHost())
}

func (je *jaegerGRPCDataSender) GenConfigYAMLStr() string {
	return fmt.Sprintf(`
  jaeger:
    protocols:
      grpc:
        endpoint: "%s"`, je.GetEndpoint())
}

func (je *jaegerGRPCDataSender) ProtocolName() string {
	return "jaeger"
}
