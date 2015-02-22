
int main() {
    int x;
    int y[5];
    int *p;
    p = &y[6];
    *p = 5;
    x = 10;
    y[3] = 5;
    return x - y[3] - *p;
}
