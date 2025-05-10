module github.com/mopeyjellyfish/hookr/testdata/simple

go 1.24.2

require github.com/mopeyjellyfish/hookr v1.2.1

// Use the local module instead of remote version
replace github.com/mopeyjellyfish/hookr => ../..

require (
	github.com/philhofer/fwd v1.1.3-0.20240916144458-20a13a1f6b7c // indirect
	github.com/tinylib/msgp v1.2.5 // indirect
)
