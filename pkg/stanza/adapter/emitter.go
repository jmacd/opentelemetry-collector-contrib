// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adapter // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"

import (
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
)

// Deprecated [v0.100.0] Use helper.LogEmitter directly instead
type LogEmitter = helper.LogEmitter

// Deprecated [v0.100.0] Use helper.NewLogEmitter directly instead
func NewLogEmitter(logger *zap.SugaredLogger, opts ...helper.EmitterOption) *LogEmitter {
	return helper.NewLogEmitter(logger, opts...)
}
