cd ~/src/opentelemetry/collector-contrib/testbed && (cd .. && make otelcontribcol-testbed) && killall otelcontribcol_testbed_darwin_arm64 && (cd tests && RUN_TESTBED=1 go test -v . -run TestTrace10kSPS/OTLP-gRPC)

TEST_ARGS="-run TestTrace10kSPS/OTLP" ./runtests.sh 
