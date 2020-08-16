TAG=`git describe --tags`
VERSION ?= `[ -d ".git" ] && git describe --tags || echo "0.0.0"`
LDFLAGS=-ldflags "-s -w -X main.appVersion=${VERSION}"
UPX := $(shell which upx)
BINARY="ssbak"

build = echo "\n\nBuilding $(1)-$(2)" && CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build ${LDFLAGS} -o dist/${BINARY}_${VERSION}_$(1)_$(2) \
	&& if [ "${UPX}" != "" ]; then ${UPX} -9 --brute dist/${BINARY}_${VERSION}_$(1)_$(2); fi \
	&& if [ $(1) = "windows" ]; then mv dist/${BINARY}_${VERSION}_$(1)_$(2) dist/${BINARY}_${VERSION}_$(1)_$(2).exe; fi

build: *.go go.*
	CGO_ENABLED=0 go build ${LDFLAGS} -o ${BINARY}
	@if [ "${UPX}" != "" ]; then ${UPX} -9 ${BINARY}; fi
	rm -rf /tmp/go-*

clean:
	rm -f ${BINARY}

release:
	mkdir -p dist
	rm -f dist/${BINARY}_${VERSION}_*
	$(call build,linux,amd64)
	$(call build,linux,386)
	$(call build,linux,arm)
	$(call build,linux,arm64)
	$(call build,darwin,amd64)
	$(call build,darwin,386)
	$(call build,windows,386)
	$(call build,windows,amd64)
