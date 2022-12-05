module github.com/shivas/abyss-blackbox

go 1.16

require (
	github.com/disintegration/gift v1.2.1
	github.com/disintegration/imaging v1.6.2
	github.com/lxn/walk v0.0.0-20210112085537-c389da54e794
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	golang.org/x/exp v0.0.0-20221126150942-6ab00d035af9
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410 // indirect
	golang.org/x/sys v0.1.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/Knetic/govaluate.v3 v3.0.0 // indirect
)

replace (
	github.com/lxn/walk => github.com/shivas/walk v0.0.0-20210212094857-c0397ac21ff8
	github.com/lxn/win => github.com/shivas/win v0.0.0-20210625114026-3e83f2d215c6
)
