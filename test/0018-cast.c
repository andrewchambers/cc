
int main() {
	long int y;
	long int *p;
	y = (long int)&y;
	p = (long int *)y;
	*p = 0;
	return (int)y;
}