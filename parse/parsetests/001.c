// Test various forms of definitions.
a, *b;
int c, *h;
static int *d;
long int e;
long long int f;
short int g;

struct foo {
	int a;
	int *b;
} h;

int (*funcptr)();
int funcdecl();
