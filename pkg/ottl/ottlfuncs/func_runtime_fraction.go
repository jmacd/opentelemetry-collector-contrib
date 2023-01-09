// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

// Modeled on
// https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route_components.proto#envoy-v3-api-field-config-route-v3-routematch-runtime-fraction

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type runtimeFraction[K any] struct {
	intnFunc func(N int) int
}

func RuntimeFraction[K any](num, denom int) (ottl.ExprFunc[K], error) {
	if num < 1 {
		return nil, fmt.Errorf("runtime fraction numerator must be >= 1: %d", num)
	}
	if denom < 1 {
		return nil, fmt.Errorf("runtime fraction denominator must be >= 1: %d", denom)
	}
	if num > denom {
		return nil, fmt.Errorf("runtime fraction numerator must be <= denominator: %d, %d", num, denom)
	}
	rf := runtimeFraction[K]{
		intnFunc: rand.New(rand.NewSource(rand.Int63())).Intn,
	}
	return rf.evaluate(num, denom), nil
}

func (rf runtimeFraction[K]) evaluate(num, denom int) ottl.ExprFunc[K] {
	return func(_ context.Context, _ K) (interface{}, error) {
		// The 1+ below places the value in range [1, N]
		x := 1 + rf.intnFunc(denom)
		return num >= x, nil
	}
}
