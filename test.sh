set -e
go test ./...
echo "testing x64cc"
go install github.com/andrewchambers/cc/cmd/x64cc
for T in `echo ./test/*.c | sort` ;
do
	x64cc -o $T.s $T
	gcc $T.s -o $T.bin
	if timeout 5s $T.bin ; then
		echo $T OK
	else 
		echo $T FAIL
	fi
done