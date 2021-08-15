SET GO111MODULE=on
rsrc -manifest gomics.exe.manifest -arch amd64 -ico gomics.ico -o gomics.syso
go generate
go build -ldflags "-s -w -H=windowsgui"
upx gomics.exe