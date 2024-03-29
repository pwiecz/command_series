module github.com/pwiecz/command_series/tools/browser

go 1.18

require (
	github.com/adrg/sysfont v0.1.2
	github.com/go-gl/gl v0.0.0-20211210172815-726fda9656d6
	github.com/inkyblackness/imgui-go/v4 v4.6.0
	github.com/pwiecz/command_series v0.0.0-20230328071614-f68d30d9469b
	github.com/pwiecz/go-fltk v0.0.0-20230328095837-266ac2ca0714
)

require (
	github.com/adrg/strutil v0.2.2 // indirect
	github.com/adrg/xdg v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20221126150942-6ab00d035af9 // indirect
)

replace github.com/pwiecz/command_series v0.0.0-20230328071614-f68d30d9469b => ../..

replace github.com/pwiecz/go-fltk v0.0.0-20230328095837-266ac2ca0714 => ../../../go-fltk
