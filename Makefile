.PHONY: proto-lint proto-generate

proto-lint:
	buf lint

proto-generate:
	buf generate
