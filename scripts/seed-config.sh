#!/usr/bin/env bash
set -euo pipefail

# 简介：
# 使用本地 docker-compose 启动的 Nacos 与 Etcd，
# 将仓库根目录下的 config.yaml 注入到：
# - Nacos: dataId=CONFIG_DATA_ID, group=CONFIG_GROUP
# - Etcd:  key=ETCD_KEY

ROOT_DIR="$(cd "$(dirname "$0")"/.. && pwd)"
CONFIG_FILE="${CONFIG_FILE:-$ROOT_DIR/config.yaml}"

# Nacos 默认参数
NACOS_HOST="${NACOS_HOST:-127.0.0.1}"
NACOS_PORT="${NACOS_PORT:-8848}"
CONFIG_GROUP="${CONFIG_GROUP:-DEFAULT_GROUP}"
CONFIG_DATA_ID="${CONFIG_DATA_ID:-config.yaml}"

# Etcd 默认参数
ETCD_ENDPOINT="${ETCD_ENDPOINT:-http://127.0.0.1:2379}"
ETCD_KEY="${ETCD_KEY:-/config-loader/config.yaml}"

if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "[seed] 未找到配置文件: $CONFIG_FILE" >&2
  exit 1
fi

echo "[seed] 使用配置文件: $CONFIG_FILE"

wait_for_http() {
  local url="$1"
  local name="$2"
  for i in {1..60}; do
    code=$(curl -s -o /dev/null -w "%{http_code}" "$url" || true)
    if [[ "$code" == "200" || "$code" == "302" || "$code" == "401" ]]; then
      echo "[seed] $name 就绪 (code=$code)"
      return 0
    fi
    echo "[seed] 等待 $name ... (尝试 $i/60)"
    sleep 2
  done
  echo "[seed] 等待 $name 超时: $url" >&2
  return 1
}

echo "[seed] 等待 Nacos 可用..."
wait_for_http "http://${NACOS_HOST}:${NACOS_PORT}/nacos/" "nacos"

echo "[seed] 向 Nacos 发布配置: dataId=$CONFIG_DATA_ID, group=$CONFIG_GROUP"
nc_result=$(curl -sS -X POST \
  "http://${NACOS_HOST}:${NACOS_PORT}/nacos/v1/cs/configs" \
  --data-urlencode "dataId=${CONFIG_DATA_ID}" \
  --data-urlencode "group=${CONFIG_GROUP}" \
  --data-urlencode "content@${CONFIG_FILE}")
if [[ "$nc_result" != "true" ]]; then
  echo "[seed] Nacos 发布失败，返回: $nc_result" >&2
  exit 1
fi
echo "[seed] Nacos 发布成功"

echo "[seed] 等待 Etcd 可用..."
wait_for_http "${ETCD_ENDPOINT}/version" "etcd"

echo "[seed] 向 Etcd 写入键: $ETCD_KEY"
KEY_B64=$(printf "%s" "$ETCD_KEY" | base64 | tr -d '\n')
VAL_B64=$(cat "$CONFIG_FILE" | base64 | tr -d '\n')
etcd_resp=$(curl -sS -X POST "${ETCD_ENDPOINT}/v3/kv/put" -H "Content-Type: application/json" -d "{\"key\":\"${KEY_B64}\",\"value\":\"${VAL_B64}\"}")
if [[ -z "$etcd_resp" ]]; then
  echo "[seed] Etcd 写入失败" >&2
  exit 1
fi
echo "[seed] Etcd 写入成功"

echo "[seed] 完成。"