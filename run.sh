cd $GOPATH/src/github.com/edgexfoundry-holding/device-video-analytics-go

# Alternately use go run main.go or launch the binary from 'make build'
#go run main.go  \
./device-video-analytics-go \
 -registry \
 -source http://localhost:8080 \
 -interval 30 \
 -scanduration 15s
