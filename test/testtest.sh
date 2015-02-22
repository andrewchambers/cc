for T in `ls *.c`
do
	gcc $T -o ./$T.gcc.bin
	if ! ./$T.gcc.bin ; then
		echo "$T failed."
		exit 1
	fi
done