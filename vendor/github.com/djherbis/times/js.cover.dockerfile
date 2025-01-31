FROM golang:1.17

RUN curl -sL https://deb.nodesource.com/setup_17.x | bash
RUN apt-get install --yes nodejs

WORKDIR /go/src/github.com/djherbis/times
COPY . .

RUN GO111MODULE=auto GOOS=js GOARCH=wasm go test -covermode=count -coverprofile=profile.cov -exec="$(go env GOROOT)/misc/wasm/go_js_wasm_exec"
