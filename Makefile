.PHONY: generate
generate:
	./scripts/generate

.PHONY: test
test:
	go test ./...

.PHONY: nats-test
nats-test:
	docker-compose -f .docker/nats-test.yml up --build --abort-on-container-exit --remove-orphans; \
	docker-compose -f .docker/nats-test.yml down --remove-orphans

.PHONY: nats-bench
nats-bench:
	docker-compose -f .docker/nats-bench.yml up --build --abort-on-container-exit --remove-orphans; \
	docker-compose -f .docker/nats-bench.yml down --remove-orphans

.PHONY: mongo-test
mongo-test:
	docker-compose -f .docker/mongo-test.yml up --build --abort-on-container-exit --remove-orphans; \
	docker-compose -f .docker/mongo-test.yml down --remove-orphans

.PHONY: coverage
coverage:
	docker-compose -f .docker/coverage.yml up --build --abort-on-container-exit --remove-orphans; \
	docker-compose -f .docker/coverage.yml down --remove-orphans; \
	go tool cover -html=out/coverage.out

.PHONY: github-test
github-test:
	docker-compose -f .docker/github.yml up --build --abort-on-container-exit --remove-orphans && \
	docker-compose -f .docker/github.yml down --remove-orphans; \

.PHONY: bench
bench:
	go test -bench=${bench} -run=${run} -count=${count} ./...

.PHONY: cli
cli:
	go install ./cmd/goes

.PHONY: mock-cli-connector
mock-cli-connector:
	go run ./internal/cmd/cli-connector/main.go

.PHONY: cli-connector
