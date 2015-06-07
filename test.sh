set -e
go test ./...
go install github.com/andrewchambers/cc/cmd/x64cc
cd test
go run runner.go
