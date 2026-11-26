module acovia.net/mscan

go 1.27.4

require (
	acovia.net/record v0.0.0
	github.com/fovlin/opt v0.0.0-20261122032552-2536ee62f8b5
	"acovia.net/mscan/scan" v0.0.0
	"acovia.net/mscan/ping" v0.0.0
)

replace acovia.net/record => ../record
replace "acovia.net/mscan/scan" => ../scan
replace "acovia.net/mscan/ping" => ../ping
