
int x;
int arr[20];
int *p = &x;
int **p2 = &p;

int main() {
    x = 1;
    arr[2] = 1;
    **p2 = x - arr[2];
    return x;
}
