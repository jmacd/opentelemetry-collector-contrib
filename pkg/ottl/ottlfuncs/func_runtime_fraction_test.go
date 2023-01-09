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

package ottlfuncs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_runtimeFraction(t *testing.T) {
	t.Run("good_cfg", func(t *testing.T) {
		type testCase struct {
			N, D int
		}

		for _, test := range []testCase{
			{
				N: 1,
				D: 100,
			},
			{
				N: 50,
				D: 100,
			},
			{
				N: 1,
				D: 2,
			},
			{
				N: 1,
				D: 3,
			},
			{
				N: 3,
				D: 7,
			},
		} {
			fake := runtimeFraction[interface{}]{
				intnFunc: func() func(int) int {
					var counter int

					return func(d int) int {
						assert.Equal(t, d, test.D)
						r := counter % d
						counter++
						return r
					}
				}(),
			}
			const reps = 1000
			ctx := context.Background()
			expr := fake.evaluate(test.N, test.D)
			success := 0
			for i := 0; i < reps*test.D; i++ {
				bexp, err := expr(ctx, nil)
				assert.NoError(t, err)
				if bexp.(bool) {
					success++
				}
			}
			assert.Equal(t, reps*test.N, success)
		}
	})

	t.Run("bad_cfg", func(t *testing.T) {
		type testCase struct {
			N, D int
			E    string
		}

		for _, test := range []testCase{
			{
				N: 0,
				D: 100,
				E: "fraction numerator must be >= 1: 0",
			}, {
				N: 1,
				D: 0,
				E: "fraction denominator must be >= 1: 0",
			}, {
				N: 2,
				D: 1,
				E: "fraction numerator must be <= denominator: 2, 1",
			}, {
				N: -2,
				D: -1,
				E: "fraction numerator must be >= 1: -2",
			},
		} {
			_, err := RuntimeFraction[interface{}](test.N, test.D)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.E)
		}
	})
}
