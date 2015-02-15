

int foo() {

}

int bar() {
	return 0 + 1 - 2 * 4 / 4 % 5 >> 6 << 7 | 8 ^ 9 & 10 || 11 && 12;
}


int test() {
	label:
	if (x + 2) {
		foo = bar;
	} else {
		x + 2;
	}
	for (;1;2) {

	}
	while(true)
		foo();

	do { 
		x + 1;
	} while(0);
	goto label;

	return 0;
}