test:
	bash -c 'diff -u <(echo -n) <(gofmt -s -d .)'
	go vet ./...
	go test ./...

snapshot-v1:
	cd .. && go run ./cmd/snapshot-v1

.PHONY: test snapshot-v1
