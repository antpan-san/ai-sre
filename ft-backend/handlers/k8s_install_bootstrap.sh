#!/usr/bin/env bash
# OpsFleet K8s 离线安装引导：目标控制机无需预装 ai-sre，仅需 bash + python3；拉 zip 与解压由 python3 标准库完成。
# 用法：curl -fsSL '<publicApiBase>/api/k8s/deploy/bootstrap.sh' | sudo bash -s -- 'ofpk8s1.…'
set -euo pipefail

REF="${1:-}"
if [[ -z "$REF" ]]; then
  echo "用法: curl -fsSL '<控制台>/api/k8s/deploy/bootstrap.sh' | sudo bash -s -- 'ofpk8s1.…'" >&2
  echo "说明: 无需预装 ai-sre；需要 python3（Ubuntu 24.04 通常已安装）。整段 ofpk8s1 引用请勿截断。" >&2
  exit 1
fi

if [[ "$(id -u)" -ne 0 ]]; then
  echo "请使用 root，或通过 sudo 执行上述 curl 管道命令。" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "未找到 python3。请先安装: apt-get update && apt-get install -y python3" >&2
  exit 1
fi

export OPSFLEET_K8S_INSTALL_REF="$REF"
exec python3 -u - <<'PY'
import base64
import json
import os
import shutil
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request


def die(msg):
    print(msg, file=sys.stderr)
    sys.exit(1)


def main():
    ref = os.environ.get("OPSFLEET_K8S_INSTALL_REF", "").strip()
    if not ref:
        die("内部错误：未设置 OPSFLEET_K8S_INSTALL_REF")

    prefix = "ofpk8s1."
    if not ref.startswith(prefix):
        die("安装引用格式错误：须为控制台生成的整段 ofpk8s1.…")

    b64 = ref[len(prefix) :]
    pad = "=" * ((4 - len(b64) % 4) % 4)
    try:
        raw = base64.urlsafe_b64decode(b64 + pad)
        data = json.loads(raw.decode("utf-8"))
    except Exception as e:
        die("解码安装引用失败: %s" % e)

    api_base = str(data.get("b") or "").rstrip("/")
    invite_id = str(data.get("i") or "").strip()
    token = str(data.get("t") or "").strip()
    if not api_base or not invite_id or not token:
        die("安装引用内容不完整（缺少 API 基址、资源 ID 或 token）")

    zip_url = "%s/api/k8s/deploy/bundle-invite/%s/zip" % (api_base, invite_id)
    req = urllib.request.Request(zip_url)
    req.add_header("X-Opsfleet-Bundle-Token", token)

    tdir = tempfile.mkdtemp(prefix="opsfleet-k8s-bootstrap-")
    try:
        try:
            with urllib.request.urlopen(req, timeout=900) as resp:
                if resp.status != 200:
                    die("下载离线包失败: HTTP %s" % resp.status)
                body = resp.read()
        except urllib.error.HTTPError as e:
            err = e.read().decode("utf-8", "replace")[:2048]
            die("下载离线包失败: HTTP %s %s" % (e.code, err))

        zip_path = os.path.join(tdir, "bundle.zip")
        with open(zip_path, "wb") as f:
            f.write(body)

        import zipfile

        out = os.path.join(tdir, "out")
        os.makedirs(out, exist_ok=True)
        with zipfile.ZipFile(zip_path) as zf:
            zf.extractall(out)

        root = out
        install_sh = os.path.join(root, "install.sh")
        if not os.path.isfile(install_sh):
            subs = [os.path.join(root, name) for name in os.listdir(root)]
            dirs = [p for p in subs if os.path.isdir(p)]
            if len(dirs) == 1:
                cand = os.path.join(dirs[0], "install.sh")
                if os.path.isfile(cand):
                    root = dirs[0]
                    install_sh = cand
        if not os.path.isfile(install_sh):
            die("离线包内未找到 install.sh")

        last_snap = "/var/lib/opsfleet-k8s/last-bundle"
        try:
            os.makedirs("/var/lib/opsfleet-k8s", exist_ok=True)
            try:
                os.chmod("/var/lib/opsfleet-k8s", 0o700)
            except Exception:
                pass
            if os.path.isdir(last_snap):
                shutil.rmtree(last_snap)
            shutil.copytree(root, last_snap, symlinks=True)
            print(
                "=== 已同步离线包至 %s（日后清理: sudo ai-sre uninstall k8s，无需控制台 id）==="
                % last_snap
            )
        except Exception as e:
            print("WARN: 无法写入 %s: %s" % (last_snap, e), file=sys.stderr)

        os.chdir(root)
        try:
            st_path = "/var/lib/opsfleet-k8s/install-ref"
            os.makedirs(os.path.dirname(st_path), exist_ok=True)
            with open(st_path, "w", encoding="utf-8") as sf:
                sf.write(ref.strip() + "\n")
        except Exception:
            pass

        p = subprocess.run(["bash", "install.sh"], check=False)
        if p.returncode != 0:
            print(
                "\n安装未完成。若需按控制台「节点配置」中的全部 master/worker 清理 K8s/etcd 残留"
                "（须已对各节点 root 免密，与 install.sh 相同）：\n"
                "  sudo ai-sre uninstall k8s\n"
                "  或: sudo ai-sre k8s cleanup %r\n"
                "（须已安装 ai-sre；将重新拉包并执行 pre_cleanup；引用须在有效期内。）\n"
                % (ref,),
                file=sys.stderr,
            )
        raise SystemExit(p.returncode)
    finally:
        shutil.rmtree(tdir, ignore_errors=True)


if __name__ == "__main__":
    main()
PY
