all:
	@go build ./cmd/taskmaster/ && ./taskmaster -c ./configs/config-subj.yaml
