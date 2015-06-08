set -e
go test ./...
go install github.com/andrewchambers/cc/cmd/x64cc
go run runner.go
