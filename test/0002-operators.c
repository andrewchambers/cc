
int x;

// TODO: 
// Should we generate test cases for these?
// Should we do table based testing?
// Test prec.
// Test all types.
// Test short circuits.

int main() {
	x = 0;
	x = x + 2;        // 2
	x = x - 1;        // 1
	x = x * 6;        // 6
	x = x / 2;        // 3
	x = x % 2;        // 1
	x = x << 2;       // 4
	x = x >> 1;       // 2
	x = x | 255;      // 255
	x = x & 3;        // 3
	x = x ^ 1;        // 2
	x = -x;           // -2
	x = x + !!x;      // -1
	x = x + (x > 2);  // -1
	x = x + (x < 2);  // 0
	// XXX <= >= != ==
    return x;
}
