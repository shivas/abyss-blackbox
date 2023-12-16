module github.com/shivas/abyss-blackbox

go 1.21

require (
	github.com/disintegration/gift v1.2.1
	github.com/disintegration/imaging v1.6.2
	github.com/lxn/walk v0.0.0-20210112085537-c389da54e794
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/shivas/go-windows-hlp v0.1.0
	golang.org/x/sys v0.15.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/google/go-cmp v0.6.0 // indirect
	golang.org/x/image v0.14.0 // indirect
	gopkg.in/Knetic/govaluate.v3 v3.0.0 // indirect
)

replace (
	github.com/lxn/walk => github.com/shivas/walk v0.0.0-20210212094857-c0397ac21ff8
	github.com/lxn/win => github.com/shivas/win v0.0.0-20210625114026-3e83f2d215c6
)
