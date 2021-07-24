SET GO111MODULE=on
rsrc -manifest gomics.exe.manifest -arch amd64 -ico gomics.ico -o gomics.syso
go get -u github.com/lu4p/binclude/cmd/binclude
go generate
REM go build -ldflags "-s -w -H=windowsgui"
go build -ldflags "-s -w"
REM upx gomics.exe