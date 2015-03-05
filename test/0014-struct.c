
struct s {
    int x;
    int y;
};

int main() {
    struct s v;
    v.x = 1;
    v.y = 1;
    return v.x - v.y;
}
