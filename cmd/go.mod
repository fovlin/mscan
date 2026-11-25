module acovia.net/mscan

go 1.27.4

require (
	acovia.net/mscan/ping v0.0.0
	acovia.net/mscan/scan v0.0.0
	acovia.net/record v0.0.0
	github.com/fovlin/opt v0.0.0-20261122032552-2536ee62f8b5
)

replace (
	acovia.net/mscan/ping => ../ping
	acovia.net/mscan/scan => ../scan
	acovia.net/record => ../../mc-saver/record
)
