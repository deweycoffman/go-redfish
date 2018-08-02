FROM golang:latest AS builder

# Download and install the latest release of dep
#ADD https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 /usr/bin/dep
#RUN chmod +x /usr/bin/dep

# Copy the code from the host and compile it
#WORKDIR $GOPATH/src/github.com/superchalupa/go-redfish
#COPY Gopkg.toml Gopkg.lock ./
#RUN dep ensure --vendor-only

# copy source tree in
RUN mkdir -p /go/src/github.com/superchalupa/go-redfish
COPY . /go/src/github.com/superchalupa/go-redfish/.

# create a self-contained build structure
RUN GOPATH=$PWD CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /app -tags simulation github.com/superchalupa/go-redfish/cmd/ec-redfish

FROM scratch
COPY --from=builder /go/src/github.com/superchalupa/go-redfish/mockup.yaml   /redfish.yaml
COPY --from=builder /go/src/github.com/superchalupa/go-redfish/mockup-logging.yaml  /redfish-logging.yaml
COPY --from=builder /app  /
EXPOSE 443
EXPOSE 8000
ENTRYPOINT ["/app"]
