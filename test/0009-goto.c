
int main() {
	start:
		goto next;
		return 1;
	success:
		return 0;
	next:
		goto success;
		return 1;
}