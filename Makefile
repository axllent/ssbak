TAG=`git describe --tags`
VERSION ?= `[ -d ".git" ] && git describe --tags || echo "0.0.0"`
LDFLAGS=-ldflags "-s -w -X github.com/axllent/ssbak/cmd.Version=${VERSION}"
BINARY=ssbak

build = echo "\n\nBuilding $(1)-$(2)" && CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build ${LDFLAGS} -o dist/${BINARY}$(3) \
	&& if [ $(1) = "windows" ]; then \
		zip -9jq dist/${BINARY}_$(1)_$(2).zip dist/${BINARY}$(3) && mv dist/${BINARY}$(3) dist/${BINARY}_$(1)_$(2)$(3); \
	else \
		tar --owner=root --group=root -C dist/ -zcf dist/${BINARY}_$(1)_$(2).tar.gz ${BINARY} && mv dist/${BINARY} dist/${BINARY}_$(1)_$(2); \
	fi

build: *.go go.*
	CGO_ENABLED=0 go build ${LDFLAGS} -o ${BINARY}
	rm -rf /tmp/go-*

clean:
	rm -f ${BINARY}

release:
	mkdir -p dist
	rm -f dist/${BINARY}_*
	$(call build,linux,amd64,)
	$(call build,linux,386,)
	$(call build,darwin,amd64,)
	$(call build,darwin,386,)
	$(call build,windows,386,.exe)
	$(call build,windows,amd64,.exe)
