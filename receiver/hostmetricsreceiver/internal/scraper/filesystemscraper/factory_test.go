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

package filesystemscraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.IsType(t, &Config{}, cfg)
}

func TestCreateMetricsScraper(t *testing.T) {
	factory := &Factory{}
	cfg := &Config{}

	scraper, err := factory.CreateMetricsScraper(context.Background(), receivertest.NewNopCreateSettings(), cfg)

	assert.NoError(t, err)
	assert.NotNil(t, scraper)
}

func TestCreateMetricsScraper_Error(t *testing.T) {
	factory := &Factory{}
	cfg := &Config{IncludeDevices: DeviceMatchConfig{Devices: []string{""}}}

	_, err := factory.CreateMetricsScraper(context.Background(), receivertest.NewNopCreateSettings(), cfg)

	assert.Error(t, err)
}
