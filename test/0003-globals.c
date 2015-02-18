
x;
*p;

int main() {
    x = 1;
    p = &x;
    *p = x - 1;
    return x;
}