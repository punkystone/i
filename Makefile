-include .env
export

hash:
	docker run -it caddy caddy hash-password
auth:
	@echo -n "$(USER):$(PASSWORD)" | base64