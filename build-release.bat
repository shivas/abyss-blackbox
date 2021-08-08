Rem rsrc -manifest ./cmd/abyss-blackbox/main.manifest -ico ./trig_96x96.ico -o ./cmd/abyss-blackbox/rsrc.syso 
Rem go build -ldflags="-H windowsgui" -o abyss-blackbox.exe
go generate ./...
go build -trimpath -ldflags="-H windowsgui -s -w" -o abyss-blackbox.exe ./cmd/abyss-blackbox 
go build -trimpath -ldflags="-s -w" ./cmd/extract/