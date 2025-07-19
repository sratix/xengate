#!/bin/sh

fyne bundle -package res -prefix Res appicon-256.png > bundled.go

fyne bundle -append -prefix Res icons/share.svg >> bundled.go
fyne bundle -append -prefix Res icons/ping.svg >> bundled.go

fyne bundle -append -prefix Res themes/default.toml >> bundled.go
fyne bundle -append -prefix Res themes/nord.toml >> bundled.go
fyne bundle -append -prefix Res themes/breeze.toml >> bundled.go

fyne bundle -append -prefix Res ../LICENSE >> bundled.go
fyne bundle -append -prefix Res licenses/BSDLICENSE >> bundled.go
fyne bundle -append -prefix Res licenses/MITLICENSE >> bundled.go
