## Background

OpenTelemetry has an existing specification for conveying sampling 
probability, [here](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/tracestate-probability-sampling.md).

The existing specification works in an environment where TraceIDs are
NOT RANDOM.  It supports only power-of-two probabilities.  We can
interpolate, but not for a tail sampler.

However, with aligned interest from the W3C tracecontext group, we
have introduced a new "random" trace flag which indicates that the
least significant 7 bytes of the TraceID are truly random.  This is
compatible with the AWS X-Ray TraceID format, [described
here](https://github.com/w3c/trace-context/issues/467).

## Objective

To support non-power of two sampling in a tail sampler, we can assume
the new random flag is present.  The existing specification uses a
"p-value" in the OpenTelemetry `tracestate`, which is really a base-2
logarithm.  A nice property of this encoding is that someone who knows
the encoding can easily convert the encoding into a sampling
probability.  For example `p:2` indicates a sampling probability of
`2**-2` or 25%.

Project requirements:

1. Support non-power-of-two sampling probability in a tail sampler.
2. Exact representation means no loss from converting to and from
   floating point representation.
3. Someone who knows the encoding can easily convert the encoding into
   a sampling probability.

## Examples worked

Take the probability 0.9 with 1 hex-digit of precision:

```
P as string:      0.9
P   in (0,1] dec: 0.90000000000000
P   in (0,1] hex: 0x1.ccccccccccccccp-01
P+1 in (1,2] hex: 0x1.e6666666666666p+00
Rounded threshold: e6666666666665 (Below)
Encoded t-value:  e-0
T-value to 7-bytes: e0000000000000
7-bytes to float:         0.875000 (inv 1.142857)
Encoded t-value:  f-0
T-value to 7-bytes: f0000000000000
7-bytes to float:         0.937500 (inv 1.066667)
```

Probability 0.9 with 2 hex digits:

```
P as string:      0.9
P   in (0,1] dec: 0.90000000000000
P   in (0,1] hex: 0x1.ccccccccccccccp-01
P+1 in (1,2] hex: 0x1.e6666666666666p+00
Rounded threshold: e6666666666665 (Below)
Encoded t-value:  e6-0
T-value to 7-bytes: e6000000000000
7-bytes to float:         0.898438 (inv 1.113043)
Encoded t-value:  e7-0
T-value to 7-bytes: e7000000000000
7-bytes to float:         0.902344 (inv 1.108225)
```

Probability 0.75 has an exact representation 

```
P as string:      0.75
P   in (0,1] dec: 0.75000000000000
P   in (0,1] hex: 0x1.80000000000000p-01
P+1 in (1,2] hex: 0x1.c0000000000000p+00
Rounded threshold: bfffffffffffff (Exact)
Encoded t-value:  b-0
T-value to 7-bytes: b0000000000000
7-bytes to float:         0.687500 (inv 1.454545)
Encoded t-value:  c-0
T-value to 7-bytes: c0000000000000
7-bytes to float:         0.750000 (inv 1.333333)

P as string:      0.75
P   in (0,1] dec: 0.75000000000000
P   in (0,1] hex: 0x1.80000000000000p-01
P+1 in (1,2] hex: 0x1.c0000000000000p+00
Rounded threshold: bfffffffffffff (Exact)
Encoded t-value:  bf-0
T-value to 7-bytes: bf000000000000
7-bytes to float:         0.746094 (inv 1.340314)
Encoded t-value:  c-0
T-value to 7-bytes: c0000000000000
7-bytes to float:         0.750000 (inv 1.333333)
```

Here is 2/3

```
P as string:      2/3
P   in (0,1] dec: 0.66666666666667
P   in (0,1] hex: 0x1.55555555555555p-01
P+1 in (1,2] hex: 0x1.aaaaaaaaaaaaaap+00
Rounded threshold: aaaaaaaaaaaaa9 (Below)
Encoded t-value:  a-0
T-value to 7-bytes: a0000000000000
7-bytes to float:         0.625000 (inv 1.600000)
Encoded t-value:  b-0
T-value to 7-bytes: b0000000000000
7-bytes to float:         0.687500 (inv 1.454545)

P as string:      2/3
P   in (0,1] dec: 0.66666666666667
P   in (0,1] hex: 0x1.55555555555555p-01
P+1 in (1,2] hex: 0x1.aaaaaaaaaaaaaap+00
Rounded threshold: aaaaaaaaaaaaa9 (Below)
Encoded t-value:  aa-0
T-value to 7-bytes: aa000000000000
7-bytes to float:         0.664062 (inv 1.505882)
Encoded t-value:  ab-0
T-value to 7-bytes: ab000000000000
7-bytes to float:         0.667969 (inv 1.497076)
```

## Table of examples

| Prob     | Width=2       | Width=3        | Width=4         | Width=5          |
|----------|---------------|----------------|-----------------|------------------|
| 0.900000 | e6-0 (0.260%) | e66-0 (0.016%) | e666-0 (0.001%) | e6666-0 (0.000%) |
| 0.750000 | bf-0 (0.524%) | bff-0 (0.033%) | bfff-0 (0.002%) | bffff-0 (0.000%) |
| 0.666667 | aa-0 (0.392%) | aaa-0 (0.024%) | aaaa-0 (0.002%) | aaaaa-0 (0.000%) |
| 0.500000 | 7f-0 (0.787%) | 7ff-0 (0.049%) | 7fff-0 (0.003%) | 7ffff-0 (0.000%) |
| 0.333333 | 55-0 (0.775%) | 555-0 (0.049%) | 5555-0 (0.003%) | 55555-0 (0.000%) |
| 0.033333 | 88-1 (0.392%) | 888-1 (0.024%) | 8888-1 (0.002%) | 88888-1 (0.000%) |
| 0.003333 | da-2 (0.250%) | da7-2 (0.021%) | da74-2 (0.002%) | da74-2 (0.000%)  |
| 0.000333 | 15-2 (4.025%) | 15d-2 (0.151%) | 15d8-2 (0.011%) | 15d86-2 (0.001%) |
| 0.000001 | 1-4 (4.858%)  | 10c-4 (0.210%) | 10c6-4 (0.023%) | 10c6f-4 (0.001%) |
