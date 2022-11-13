module github.com/pwiecz/command_series/tools/browser

go 1.18

require (
	github.com/adrg/sysfont v0.1.2
	github.com/go-gl/gl v0.0.0-20211210172815-726fda9656d6
	github.com/inkyblackness/imgui-go/v4 v4.6.0
	github.com/pwiecz/command_series v0.0.0-20221106220632-4c51ce240179
	github.com/pwiecz/go-fltk v0.0.0-20221107213751-8a22d4c44b95
)

require (
	github.com/adrg/strutil v0.2.2 // indirect
	github.com/adrg/xdg v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20220722155223-a9213eeb770e // indirect
)

//replace github.com/pwiecz/command_series v0.0.0-20221106220632-4c51ce240179 => ../..
//replace github.com/pwiecz/go-fltk v0.0.0-20221107213751-8a22d4c44b95 => ../../../go-fltk
