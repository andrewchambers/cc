set -e
go test ./...
echo "testing x64cc"
go build ./cmd/x64cc
for T in `echo ./test/*.c | sort` ;
do
	./x64cc -o $T.s $T
	gcc $T.s -o $T.bin
	timeout 5s $T.bin
	echo $T OK
done