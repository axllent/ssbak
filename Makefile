TAG=`git describe --tags`
VERSION ?= `[ -d ".git" ] && git describe --tags || echo "0.0.0"`
LDFLAGS=-ldflags "-s -w -X github.com/axllent/ssbak/cmd.Version=${VERSION}"
BINARY="ssbak"

build = echo "\n\nBuilding $(1)-$(2)" && CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build ${LDFLAGS} -o dist/${BINARY}_$(1)_$(2) \
	&& if [ $(1) = "windows" ]; then mv dist/${BINARY}_$(1)_$(2) dist/${BINARY}_$(1)_$(2).exe; fi

build: *.go go.*
	CGO_ENABLED=0 go build ${LDFLAGS} -o ${BINARY}
	rm -rf /tmp/go-*

clean:
	rm -f ${BINARY}

release:
	mkdir -p dist
	rm -f dist/${BINARY}_*
	$(call build,linux,amd64)
	$(call build,linux,386)
	$(call build,darwin,amd64)
	$(call build,darwin,386)
	$(call build,windows,386)
	$(call build,windows,amd64)
