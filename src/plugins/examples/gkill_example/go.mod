module github.com/mt3hr/gkill_example

go 1.26.6

require github.com/mt3hr/gkill/src/server v0.0.0

require (
	github.com/mattn/go-zglob v0.0.6 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/mt3hr/gkill/src/server => ../../../server
