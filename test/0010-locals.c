
int main() {
    int x;
    int *p;
    int y[10];
    p = &y[6];
    *p = 5;
    x = 10;
    y[3] = 5;
    return x - y[3] - *p;
}
