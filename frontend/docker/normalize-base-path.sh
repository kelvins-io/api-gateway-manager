#!/bin/sh
# 与 frontend/vite.config.ts 的 normalizeBasePath 保持一致。
normalize_base_path() {
  raw=$(printf '%s' "${1:-}" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  if [ -z "$raw" ] || [ "$raw" = "/" ]; then
    printf '/'
    return 0
  fi
  case "$raw" in
    /*) ;;
    *) raw="/$raw" ;;
  esac
  case "$raw" in
    */) raw=${raw%/} ;;
  esac
  case "$raw" in
    *//*)
      echo "无效的 VITE_BASE_PATH: $1" >&2
      return 1
      ;;
  esac
  case "$raw" in
    /[A-Za-z0-9/_-]*) ;;
    *)
      echo "无效的 VITE_BASE_PATH: $1" >&2
      return 1
      ;;
  esac
  # 拒绝路径段以外的字符（case 只检查开头）
  rest=$(printf '%s' "$raw" | sed 's|^/||;s|[A-Za-z0-9/_-]||g')
  if [ -n "$rest" ]; then
    echo "无效的 VITE_BASE_PATH: $1" >&2
    return 1
  fi
  printf '%s/' "$raw"
}
