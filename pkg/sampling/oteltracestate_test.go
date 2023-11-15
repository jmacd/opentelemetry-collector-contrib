// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampling

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testName(in string) string {
	if len(in) > 32 {
		return in[:32] + "..."
	}
	return in
}

func TestEmptyOTelTraceState(t *testing.T) {
	// Empty value is invalid
	_, err := NewOTelTraceState("")
	require.Error(t, err)
}

func TestOTelTraceStateTValueSerialize(t *testing.T) {
	const orig = "r:10000000000000;t:3;a:b;c:d"
	otts, err := NewOTelTraceState(orig)
	require.NoError(t, err)
	require.True(t, otts.HasTValue())
	require.Equal(t, "3", otts.TValue())
	require.Equal(t, 1-0x3p-4, otts.TValueThreshold().Probability())

	require.True(t, otts.HasRValue())
	require.Equal(t, "10000000000000", otts.RValue())
	require.Equal(t, "10000000000000", otts.RValueRandomness().RValue())

	require.True(t, otts.HasAnyValue())
	var w strings.Builder
	otts.Serialize(&w)
	require.Equal(t, orig, w.String())
}

func TestOTelTraceStateZero(t *testing.T) {
	const orig = "t:0"
	otts, err := NewOTelTraceState(orig)
	require.NoError(t, err)
	require.True(t, otts.HasAnyValue())
	require.True(t, otts.HasTValue())
	require.Equal(t, "0", otts.TValue())
	require.Equal(t, 1.0, otts.TValueThreshold().Probability())

	var w strings.Builder
	otts.Serialize(&w)
	require.Equal(t, orig, w.String())
}

func TestOTelTraceStateRValuePValue(t *testing.T) {
	// Ensures the caller can handle RValueSizeError and search
	// for p-value in extra-values.
	const orig = "r:3;p:2"
	otts, err := NewOTelTraceState(orig)
	require.Error(t, err)
	require.True(t, errors.Is(err, RValueSizeError("3")))
	require.False(t, otts.HasRValue())

	// The error is oblivious to the old r-value, but that's ok.
	require.Contains(t, err.Error(), "14 hex digits")

	require.Equal(t, []KV{{"p", "2"}}, otts.ExtraValues())

	var w strings.Builder
	otts.Serialize(&w)
	require.Equal(t, "p:2", w.String())
}

func TestOTelTraceStateTValueUpdate(t *testing.T) {
	const orig = "r:abcdefabcdefab"
	otts, err := NewOTelTraceState(orig)
	require.NoError(t, err)
	require.False(t, otts.HasTValue())
	require.True(t, otts.HasRValue())

	th, _ := TValueToThreshold("3")
	require.NoError(t, otts.UpdateTValueWithSampling(th, "3"))

	require.Equal(t, "3", otts.TValue())
	require.Equal(t, 1-0x3p-4, otts.TValueThreshold().Probability())

	const updated = "r:abcdefabcdefab;t:3"
	var w strings.Builder
	otts.Serialize(&w)
	require.Equal(t, updated, w.String())
}

func TestOTelTraceStateRTUpdate(t *testing.T) {
	otts, err := NewOTelTraceState("a:b")
	require.NoError(t, err)
	require.False(t, otts.HasTValue())
	require.False(t, otts.HasRValue())
	require.True(t, otts.HasAnyValue())

	th, _ := TValueToThreshold("3")
	require.NoError(t, otts.UpdateTValueWithSampling(th, "3"))
	otts.SetRValue(must(RValueToRandomness("00000000000003")))

	const updated = "r:00000000000003;t:3;a:b"
	var w strings.Builder
	otts.Serialize(&w)
	require.Equal(t, updated, w.String())
}

func TestOTelTraceStateRTClear(t *testing.T) {
	otts, err := NewOTelTraceState("a:b;r:12341234123412;t:1234")
	require.NoError(t, err)

	otts.ClearTValue()
	otts.ClearRValue()

	const updated = "a:b"
	var w strings.Builder
	otts.Serialize(&w)
	require.Equal(t, updated, w.String())
}

func TestParseOTelTraceState(t *testing.T) {
	type testCase struct {
		in        string
		rval      string
		tval      string
		extra     []string
		expectErr error
	}
	const ns = ""
	for _, test := range []testCase{
		// t-value correct cases
		{"t:2", ns, "2", nil, nil},
		{"t:1", ns, "1", nil, nil},
		{"t:1", ns, "1", nil, nil},
		{"t:10", ns, "10", nil, nil},
		{"t:33", ns, "33", nil, nil},
		{"t:ab", ns, "ab", nil, nil},
		{"t:61", ns, "61", nil, nil},

		// syntax errors
		{"", ns, ns, nil, strconv.ErrSyntax},
		{"t:1;", ns, ns, nil, strconv.ErrSyntax},
		{"t:1=p:2", ns, ns, nil, strconv.ErrSyntax},
		{"t:1;p:2=s:3", ns, ns, nil, strconv.ErrSyntax},
		{":1;p:2=s:3", ns, ns, nil, strconv.ErrSyntax},
		{":;p:2=s:3", ns, ns, nil, strconv.ErrSyntax},
		{":;:", ns, ns, nil, strconv.ErrSyntax},
		{":", ns, ns, nil, strconv.ErrSyntax},
		{"t:;p=1", ns, ns, nil, strconv.ErrSyntax},
		{"t:$", ns, ns, nil, strconv.ErrSyntax},      // not-hexadecimal
		{"t:0x1p+3", ns, ns, nil, strconv.ErrSyntax}, // + is invalid
		{"t:14.5", ns, ns, nil, strconv.ErrSyntax},   // integer syntax
		{"t:-1", ns, ns, nil, strconv.ErrSyntax},     // non-negative

		// too many digits
		{"t:ffffffffffffffff", ns, ns, nil, ErrTValueSize},
		{"t:100000000000000", ns, ns, nil, ErrTValueSize},

		// one field
		{"e100:1", ns, ns, []string{"e100:1"}, nil},

		// two fields
		{"e1:1;e2:2", ns, ns, []string{"e1:1", "e2:2"}, nil},
		{"e1:1;e2:2", ns, ns, []string{"e1:1", "e2:2"}, nil},

		// one extra key, two ways
		{"t:2;extra:stuff", ns, "2", []string{"extra:stuff"}, nil},
		{"extra:stuff;t:2", ns, "2", []string{"extra:stuff"}, nil},

		// two extra fields
		{"e100:100;t:1;e101:101", ns, "1", []string{"e100:100", "e101:101"}, nil},
		{"t:1;e100:100;e101:101", ns, "1", []string{"e100:100", "e101:101"}, nil},
		{"e100:100;e101:101;t:1", ns, "1", []string{"e100:100", "e101:101"}, nil},

		// parse error prevents capturing unrecognized keys
		{"1:1;u:V", ns, ns, nil, strconv.ErrSyntax},
		{"X:1;u:V", ns, ns, nil, strconv.ErrSyntax},
		{"x:1;u:V", ns, ns, []string{"x:1", "u:V"}, nil},

		// r-value
		{"r:22222222222222;extra:stuff", "22222222222222", ns, []string{"extra:stuff"}, nil},
		{"extra:stuff;r:22222222222222", "22222222222222", ns, []string{"extra:stuff"}, nil},
		{"r:ffffffffffffff", "ffffffffffffff", ns, nil, nil},
		{"r:88888888888888", "88888888888888", ns, nil, nil},
		{"r:00000000000000", "00000000000000", ns, nil, nil},

		// r-value range error (15 bytes of hex or more)
		{"r:100000000000000", ns, ns, nil, RValueSizeError("100000000000000")},
		{"r:fffffffffffffffff", ns, ns, nil, RValueSizeError("fffffffffffffffff")},

		// no trailing ;
		{"x:1;", ns, ns, nil, strconv.ErrSyntax},

		// empty key
		{"x:", ns, ns, []string{"x:"}, nil},

		// charset test
		{"x:0X1FFF;y:.-_-.;z:", ns, ns, []string{"x:0X1FFF", "y:.-_-.", "z:"}, nil},
		{"x1y2z3:1-2-3;y1:y_1;xy:-;t:50", ns, "50", []string{"x1y2z3:1-2-3", "y1:y_1", "xy:-"}, nil},

		// size exceeded
		{"x:" + strings.Repeat("_", 255), ns, ns, nil, ErrTraceStateSize},
		{"x:" + strings.Repeat("_", 254), ns, ns, []string{"x:" + strings.Repeat("_", 254)}, nil},
	} {
		t.Run(testName(test.in), func(t *testing.T) {
			otts, err := NewOTelTraceState(test.in)

			if test.expectErr != nil {
				require.True(t, errors.Is(err, test.expectErr), "%q: not expecting %v wanted %v", test.in, err, test.expectErr)
			} else {
				require.NoError(t, err)
			}
			if test.rval != ns {
				require.True(t, otts.HasRValue())
				require.Equal(t, test.rval, otts.RValue())
			} else {
				require.False(t, otts.HasRValue(), "should have no r-value: %s", otts.RValue())
			}
			if test.tval != ns {
				require.True(t, otts.HasTValue())
				require.Equal(t, test.tval, otts.TValue())
			} else {
				require.False(t, otts.HasTValue(), "should have no t-value: %s", otts.TValue())
			}
			var expect []KV
			for _, ex := range test.extra {
				k, v, _ := strings.Cut(ex, ":")
				expect = append(expect, KV{
					Key:   k,
					Value: v,
				})
			}
			require.Equal(t, expect, otts.ExtraValues())

			if test.expectErr != nil {
				return
			}
			// on success Serialize() should not modify
			// test by re-parsing
			var w strings.Builder
			otts.Serialize(&w)
			cpy, err := NewOTelTraceState(w.String())
			require.NoError(t, err)
			require.Equal(t, otts, cpy)
		})
	}
}

func TestUpdateTValueWithSampling(t *testing.T) {
	type testCase struct {
		// The input otel tracestate; no error conditions tested
		in string

		// The incoming adjusted count; defined whether
		// t-value is present or not.
		adjCountIn float64

		// the update probability; threshold and tvalue are
		// derived from this
		prob float64

		// when update error is expected
		updateErr error

		// output t-value
		out string

		// output adjusted count
		adjCountOut float64
	}
	for _, test := range []testCase{
		// 8/16 in, sampled at (0x10-0xe)/0x10 = 2/16 => adjCount 8
		{"t:8", 2, 0x2p-4, nil, "t:e", 8},

		// 8/16 in, sampled at 14/16 => no update, adjCount 2
		{"t:8", 2, 0xep-4, nil, "t:8", 2},

		// 1/16 in, 50% update (error)
		{"t:f", 16, 0x8p-4, ErrInconsistentSampling, "t:f", 16},

		// 1/1 sampling in, 1/16 update
		{"t:0", 1, 0x1p-4, nil, "t:f", 16},

		// no t-value in, 1/16 update
		{"", 0, 0x1p-4, nil, "t:f", 16},

		// none in, 100% update
		{"", 0, 1, nil, "t:0", 1},

		// 1/2 in, 100% update (error)
		{"t:8", 2, 1, ErrInconsistentSampling, "t:8", 2},

		// 1/1 in, 0x1p-56 update
		{"t:0", 1, 0x1p-56, nil, "t:ffffffffffffff", 0x1p56},

		// 1/1 in, 0x1p-56 update
		{"t:0", 1, 0x1p-56, nil, "t:ffffffffffffff", 0x1p56},

		// 2/3 in, 1/3 update.  Note that 0x555 + 0xaab = 0x1000.
		{"t:555", 1 / (1 - 0x555p-12), 0x555p-12, nil, "t:aab", 1 / (1 - 0xaabp-12)},
	} {
		t.Run(test.in+"/"+test.out, func(t *testing.T) {
			otts := OTelTraceState{}
			if test.in != "" {
				var err error
				otts, err = NewOTelTraceState(test.in)
				require.NoError(t, err)
			}

			require.Equal(t, test.adjCountIn, otts.AdjustedCount())

			newTh, err := ProbabilityToThreshold(test.prob)
			require.NoError(t, err)

			upErr := otts.UpdateTValueWithSampling(newTh, newTh.TValue())

			if test.updateErr != nil {
				require.Equal(t, test.updateErr, upErr)
			}

			var outData strings.Builder
			err = otts.Serialize(&outData)
			require.NoError(t, err)
			require.Equal(t, test.out, outData.String())

			require.Equal(t, test.adjCountOut, otts.AdjustedCount())
		})
	}
}
