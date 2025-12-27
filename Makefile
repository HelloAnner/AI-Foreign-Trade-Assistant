## AI Foreign Trade Assistant — Makefile
## 目标：对标 get_jobs 的多架构打包与私有仓库上传（latest），并提供双容器（Browserless+App）运行方案。
## 用法示例：
##   make upload                # 多架构 buildx + 推送到私有 Hub（latest）并校验
##   make compose-up            # 启动 browserless + app 两个容器（分析用）
##   make compose-down          # 停止并移除
##   make prune-cache           # 清理 buildx 缓存
##   make print                 # 打印当前配置
##
## 注意：完全遵循“AGENTS.md 工程文化”与“禁止 git add/commit”。

SHELL := /bin/bash

# ===== 默认配置（可用环境变量覆盖） =====
GITEA_HOST    ?= 118.196.16.11
GITEA_PORT    ?= 48080
REGISTRY      ?= $(GITEA_HOST):$(GITEA_PORT)
NAMESPACE     ?= anner
REPO          ?= $(notdir $(CURDIR))
PLATFORMS     ?= linux/amd64,linux/arm64
BUILDER_NAME  ?= $(REPO)-builder
PROGRESS      ?= $(if $(shell test -t 1 && echo tty),tty,plain)
NO_CACHE      ?=
INSECURE      ?= 1

# 推送的镜像名（仅 latest）
IMAGE := $(REGISTRY)/$(NAMESPACE)/$(REPO):latest

# compose 使用的镜像名（可通过 make 变量覆盖）
COMPOSE_IMAGE ?= $(IMAGE)

.PHONY: help print upload buildx-init buildx-build verify-latest clean-tags login prune-cache start stop logs server-start server-stop server-logs rm-scripts

help:
	@echo "make upload        # 多架构构建并推送 latest（对标 get_jobs/scripts/build-and-push.sh）" \
	&& echo "make start         # 本地基于代码的 docker compose 启动（build 或使用本地镜像）" \
	&& echo "make stop          # 本地基于代码的 docker compose 停止并清理" \
	&& echo "make logs          # 本地容器日志跟随" \
	&& echo "make server-start  # 服务器模式：仅拉取 Hub 镜像启动（无需源码）" \
	&& echo "make server-stop   # 服务器模式：停止并删除容器" \
	&& echo "make prune-cache   # 清理 buildx 缓存" \
	&& echo "make print         # 打印当前配置" \
	&& echo "make rm-scripts    # 删除 scripts 文件夹（谨慎，确认后手动执行）"

print:
	@echo "Registry   : $(REGISTRY)" \
	&& echo "Namespace  : $(NAMESPACE)" \
	&& echo "Repository : $(REPO)" \
	&& echo "Image      : $(IMAGE)" \
	&& echo "Platforms  : $(PLATFORMS)" \
	&& echo "Builder    : $(BUILDER_NAME)" \
	&& echo "Insecure   : $(INSECURE) (config buildx http=true)" \
	&& echo "Progress   : $(PROGRESS)"

# 入口：构建并推送 latest，并校验为多架构清单
upload: buildx-init buildx-build verify-latest clean-tags
	@echo "[OK] upload 完成：$(IMAGE)"

# 初始化 buildx 与 binfmt

buildx-init:
	@echo "[1/4] 初始化 buildx 与 binfmt…" \
	&& docker pull tonistiigi/binfmt:latest >/dev/null 2>&1 || true \
	&& docker run --privileged --rm tonistiigi/binfmt:latest --install all >/dev/null 2>&1 || true \
	&& if ! docker buildx inspect "$(BUILDER_NAME)" >/dev/null 2>&1; then \
		if [ "$(INSECURE)" = "1" ]; then \
			cfg=$$(mktemp); \
			printf 'debug = true\n[registry.%q]\n  http = true\n  insecure = true\n' "$(REGISTRY)" > $$cfg; \
			docker buildx create --name "$(BUILDER_NAME)" --driver docker-container --driver-opt network=host --config $$cfg >/dev/null; \
			rm -f $$cfg; \
		else \
			docker buildx create --name "$(BUILDER_NAME)" --driver docker-container >/dev/null; \
		fi; \
	fi \
	&& docker buildx use "$(BUILDER_NAME)" \
	&& docker buildx inspect --bootstrap >/dev/null

# 构建并推送 latest（多架构）
buildx-build:
	@echo "[2/4] 构建并推送：$(IMAGE)" \
	&& DOCKER_BUILDKIT=1 docker buildx build \
		--platform "$(PLATFORMS)" \
		-t "$(IMAGE)" \
		--push \
		--progress "$(PROGRESS)" \
		$(if $(NO_CACHE),--no-cache,) \
		.

# 校验 latest 是否为 manifest list 且包含 amd64+arm64

verify-latest:
	@echo "[3/4] 校验 multi-arch manifest: $(IMAGE)" \
	&& set -e; \
	accept='application/vnd.docker.distribution.manifest.list.v2+json'; \
	url="http://$(REGISTRY)/v2/$(NAMESPACE)/$(REPO)/manifests/latest"; \
	body=$$(curl -fsS -H "Accept: $$accept" "$$url" || true); \
	if [ -z "$$body" ] && [ -n "$$GITEA_USERNAME" ]; then \
		body=$$(curl -fsS -u "$$GITEA_USERNAME:$$GITEA_PASSWORD" -H "Accept: $$accept" "$$url" || true); \
	fi; \
	if [ -z "$$body" ]; then echo "WARN: 跳过校验（无法获取 manifest）。"; exit 0; fi; \
	count=$$(printf '%s' "$$body" | tr '\n' ' ' | grep -Eo '"architecture"\s*:\s*"(amd64|arm64)"' | sort -u | wc -l | awk '{print $$1}'); \
	if [ "$$count" -lt 2 ]; then echo "ERROR: latest 未同时包含 amd64 与 arm64。"; exit 1; fi; \
	echo "校验通过：包含 linux/amd64 与 linux/arm64"

# 仅保留 latest（需要 GITEA_USERNAME/PASSWORD）
clean-tags:
ifeq ($(strip $(GITEA_USERNAME)),)
ifeq ($(strip $(GITEA_PASSWORD)),)
	@echo "[4/4] 清理旧标签：跳过（未设置 GITEA_USERNAME/GITEA_PASSWORD）"
else
	@echo "[4/4] 清理旧标签：跳过（未设置 GITEA_USERNAME）"
endif
else
	@echo "[4/4] 清理旧标签：仅保留 latest"
	@set -e; \
	base_v1="http://$(GITEA_HOST):$(GITEA_PORT)/api/v1"; \
	base_v0="http://$(GITEA_HOST):$(GITEA_PORT)/api"; \
	list_url_v1="$$base_v1/packages/$(NAMESPACE)/container/$(REPO)/versions"; \
	list_url_v0="$$base_v0/packages/$(NAMESPACE)/container/$(REPO)/versions"; \
	body=$$( (curl -fsS -u "$(GITEA_USERNAME):$(GITEA_PASSWORD)" "$$list_url_v1" || curl -fsS -u "$(GITEA_USERNAME):$(GITEA_PASSWORD)" "$$list_url_v0") ); \
	versions=$$(printf '%s' "$$body" | tr -d '\r' | grep -o '"version"[[:space:]]*:[[:space:]]*"[^"]*"' | sed -E 's/.*"version"[[:space:]]*:[[:space:]]*"([^"]*)".*/\1/' | sort -u); \
	for v in $$versions; do \
		if [ "$$v" = "latest" ]; then echo "保留: latest"; continue; fi; \
		curl -fsS -u "$(GITEA_USERNAME):$(GITEA_PASSWORD)" -X DELETE "$$base_v1/packages/$(NAMESPACE)/container/$(REPO)/$$v" >/dev/null 2>&1 || \
		curl -fsS -u "$(GITEA_USERNAME):$(GITEA_PASSWORD)" -X DELETE "$$base_v0/packages/$(NAMESPACE)/container/$(REPO)/$$v" >/dev/null 2>&1 || true; \
		echo "删除: $$v"; \
	done
endif

login:
ifeq ($(strip $(DOCKER_USERNAME)),)
	@echo "跳过 docker login（未设置 DOCKER_USERNAME/DOCKER_PASSWORD）"
else
	@echo "登录 $(REGISTRY)…"
	@echo "$(DOCKER_PASSWORD)" | docker login "$(REGISTRY)" -u "$(DOCKER_USERNAME)" --password-stdin
endif

prune-cache:
	@docker buildx prune -f || true

# ===== 本地模式：基于代码的 docker compose =====
start:
	@docker compose -f docker-compose.yml up -d --remove-orphans

stop:
	@docker compose -f docker-compose.yml down -v || true; \
	docker rm -f fta-app >/dev/null 2>&1 || true; \
	docker rm -f fta-browserless >/dev/null 2>&1 || true; \
	docker network rm fta_net >/dev/null 2>&1 || true

logs:
	@docker compose -f docker-compose.yml logs -f --tail=200

# ===== 服务器模式：仅拉取 Hub 镜像启动（无源码构建依赖） =====
# 可配置变量（服务器仅持有 Makefile 与配置）：
#   SERVER_IMAGE ?= $(REGISTRY)/$(NAMESPACE)/$(REPO):latest
#   DATA_DIR     ?= ~/.foreign_trade
#   NETWORK      ?= fta_net
#   BROWSERLESS_TAG ?= ghcr.io/browserless/chromium:latest
#   BROWSERLESS_TOKEN ?= devtoken
#   BROWSERLESS_TIMEOUT ?= 600000
#   BROWSERLESS_PORT ?= 3000
#   APP_PORT ?= 25000
#   PULL_ALWAYS ?= 1
SERVER_IMAGE       ?= $(REGISTRY)/$(NAMESPACE)/$(REPO):latest
DATA_DIR           ?= ~/.foreign_trade
NETWORK            ?= fta_net
BROWSERLESS_TAG    ?= ghcr.io/browserless/chromium:latest
BROWSERLESS_TOKEN  ?= devtoken
BROWSERLESS_TIMEOUT?= 600000
BROWSERLESS_PORT   ?= 3000
APP_PORT           ?= 25000
BROWSERLESS_MODE   ?= playwright
PW_SLOWMO_MS       ?= 0
PULL_ALWAYS        ?= 1

server-start:
	@set -e; \
	if [ "$(PULL_ALWAYS)" = "1" ]; then \
	  echo "拉取镜像…"; \
	  docker pull "$(BROWSERLESS_TAG)" || true; \
	  docker pull "$(SERVER_IMAGE)" || true; \
	fi; \
	if ! docker network inspect "$(NETWORK)" >/dev/null 2>&1; then \ 
	  echo "创建网络 $(NETWORK)…"; docker network create "$(NETWORK)" >/dev/null; \ 
	fi; \
	echo "启动 browserless…"; \
	docker rm -f fta-browserless >/dev/null 2>&1 || true; \
	docker run -d --name fta-browserless \
	  --network "$(NETWORK)" \
	  --restart unless-stopped \
	  --shm-size 1g \
	  -e TOKEN="$(BROWSERLESS_TOKEN)" \
	  -e TIMEOUT="$(BROWSERLESS_TIMEOUT)" \
	  -p "$(BROWSERLESS_PORT):3000" \
	  "$(BROWSERLESS_TAG)" >/dev/null; \
	BWS="ws://fta-browserless:3000/chromium/playwright?token=$(BROWSERLESS_TOKEN)&timeout=$(BROWSERLESS_TIMEOUT)"; \
	echo "启动 app…"; \
	docker rm -f fta-app >/dev/null 2>&1 || true; \
	docker run -d --name fta-app \
	  --network "$(NETWORK)" \
	  --restart unless-stopped \
	  -e BROWSERLESS_URL="$$BWS" \
	  -e BROWSERLESS_MODE="$(BROWSERLESS_MODE)" \
	  -e PW_SLOWMO_MS="$(PW_SLOWMO_MS)" \
	  -e FTA_DATA_DIR=/data \
	  -e APP_PORT="$(APP_PORT)" \
	  -e FOREIGN_TRADE_NO_BROWSER=1 \
	  -p "$(APP_PORT):25000" \
	  -v "$(DATA_DIR):/data" \
	  "$(SERVER_IMAGE)" >/dev/null; \
	echo "Server started: http://localhost:$(APP_PORT)"

server-stop:
	@docker rm -f fta-app >/dev/null 2>&1 || true; \
	docker rm -f fta-browserless >/dev/null 2>&1 || true; \
	echo "Server stopped."

server-logs:
	@docker logs -f --tail=200 fta-app

# 谨慎使用：删除 scripts 目录。建议在确认 Makefile 替代逻辑无误后再执行。
rm-scripts:
	@rm -rf scripts
	@echo "已删除 scripts/ 目录"
