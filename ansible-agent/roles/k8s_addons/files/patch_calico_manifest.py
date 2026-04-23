#!/usr/bin/env python3
"""Patch upstream calico.yaml so it is internally consistent for VXLAN mode.

Upstream `manifests/calico.yaml` defaults to IPIP+BIRD. Just flipping
``calico_backend`` to ``vxlan`` leaves the pod/pool env vars on IPIP and,
critically, leaves ``-bird-live``/``-bird-ready`` in the liveness and
readiness probes. With BIRD disabled those probes fail forever and
``calico-node`` crashloops. This helper performs the full set of edits:

- ConfigMap ``calico_backend`` → ``vxlan`` (when requested)
- Enable the ``CALICO_IPV4POOL_CIDR`` env var with the cluster pod CIDR
- When VXLAN is requested: set ``CALICO_IPV4POOL_IPIP=Never``,
  ``CALICO_IPV4POOL_VXLAN=Always``, ``CLUSTER_TYPE=k8s,vxlan``, and drop
  ``-bird-live``/``-bird-ready`` from the calico-node probes
- Loosen calico-node felix liveness/readiness probe thresholds so
  resource-constrained lab VMs (ARM64 / nested) don't crashloop during
  felix cold-start
- Optional ConfigMap ``veth_mtu`` override

Idempotent: re-running against already-patched output is a no-op.
"""
import re
import sys


def _sub_once(text: str, pattern: str, repl: str, flags: int = 0) -> tuple[str, int]:
    new_text, n = re.subn(pattern, repl, text, count=1, flags=flags)
    return new_text, n


def main() -> None:
    if len(sys.argv) < 4:
        print(
            "Usage: patch_calico_manifest.py <in.yaml> <out.yaml> <cidr> <vxlan:true|false> [mtu]",
            file=sys.stderr,
        )
        sys.exit(1)
    in_path, out_path, cidr, vx = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
    mt = int(sys.argv[5]) if len(sys.argv) > 5 and str(sys.argv[5]).isdigit() else 0
    use_vxlan = str(vx).lower() in ("1", "true", "yes", "on")
    text = open(in_path, encoding="utf-8").read()

    if use_vxlan:
        text = text.replace('calico_backend: "bird"', 'calico_backend: "vxlan"')

    text, n = _sub_once(
        text,
        r"^\s*#\s*-\s*name:\s*CALICO_IPV4POOL_CIDR\n\s*#\s*value:\s*\"192.168.0.0/16\"",
        f"            - name: CALICO_IPV4POOL_CIDR\n              value: \"{cidr}\"",
        flags=re.MULTILINE,
    )
    if n != 1 and "CALICO_IPV4POOL_CIDR" not in text:
        print("ERROR: could not find CALICO_IPV4POOL_CIDR block to uncomment", file=sys.stderr)
        sys.exit(2)

    if use_vxlan:
        text, _ = _sub_once(
            text,
            r'(- name: CALICO_IPV4POOL_IPIP\n\s+value: )"Always"',
            r'\1"Never"',
        )
        text, _ = _sub_once(
            text,
            r'(- name: CALICO_IPV4POOL_VXLAN\n\s+value: )"Never"',
            r'\1"Always"',
        )
        text, _ = _sub_once(
            text,
            r'(- name: CLUSTER_TYPE\n\s+value: )"k8s,bgp"',
            r'\1"k8s,vxlan"',
        )
        text = re.sub(r"^\s*-\s*-bird-live\n", "", text, flags=re.MULTILINE)
        text = re.sub(r"^\s*-\s*-bird-ready\n", "", text, flags=re.MULTILINE)

    # 放宽 calico-node 探针阈值：
    # 资源紧张（嵌套/ARM64 小内存）的实验环境里，felix 冷启动可能超过上游默认 ~60s
    # 存活阈值，kubelet 会把 calico-node 容器周期性杀掉重建 sandbox，继而把
    # coredns / kube-controllers 一起拖进 CrashLoopBackOff。只放宽 -felix-live/
    # -felix-ready 探针（通过命令名定位），不影响 calico-kube-controllers 探针。
    def _loosen_felix_probe(text_in: str, marker: str) -> str:
        lines = text_in.splitlines(keepends=True)
        out: list[str] = []
        i = 0
        bumped = False
        while i < len(lines):
            ln = lines[i]
            out.append(ln)
            if not bumped and re.match(rf"^(\s+)-\s*{re.escape(marker)}\s*$", ln):
                m = re.match(r"^(\s+)-\s", ln)
                assert m is not None
                arg_indent = len(m.group(1))
                j = i + 1
                field_indent = None
                seen = {"period": False, "timeout": False, "failure": False, "initial": False}
                block_end = j
                while j < len(lines):
                    nxt = lines[j]
                    lstrip_len = len(nxt) - len(nxt.lstrip(" "))
                    if nxt.strip() == "":
                        block_end = j + 1
                        j += 1
                        continue
                    if lstrip_len >= arg_indent:
                        block_end = j + 1
                        j += 1
                        continue
                    key = re.match(r"^(\s+)(periodSeconds|timeoutSeconds|failureThreshold|initialDelaySeconds|successThreshold):", nxt)
                    if key:
                        if field_indent is None:
                            field_indent = key.group(1)
                        block_end = j + 1
                        j += 1
                        continue
                    break
                if field_indent is None:
                    field_indent = " " * (arg_indent - 4)
                new_block: list[str] = []
                for k in range(i + 1, block_end):
                    cur = lines[k]
                    if re.match(r"^\s+periodSeconds:", cur):
                        new_block.append(f"{field_indent}periodSeconds: 30\n")
                        seen["period"] = True
                    elif re.match(r"^\s+timeoutSeconds:", cur):
                        new_block.append(f"{field_indent}timeoutSeconds: 15\n")
                        seen["timeout"] = True
                    elif re.match(r"^\s+failureThreshold:", cur):
                        new_block.append(f"{field_indent}failureThreshold: 10\n")
                        seen["failure"] = True
                    elif re.match(r"^\s+initialDelaySeconds:", cur):
                        new_block.append(f"{field_indent}initialDelaySeconds: 60\n")
                        seen["initial"] = True
                    else:
                        new_block.append(cur)
                if not seen["period"]:
                    new_block.append(f"{field_indent}periodSeconds: 30\n")
                if not seen["timeout"]:
                    new_block.append(f"{field_indent}timeoutSeconds: 15\n")
                if not seen["failure"]:
                    new_block.append(f"{field_indent}failureThreshold: 10\n")
                if not seen["initial"]:
                    new_block.append(f"{field_indent}initialDelaySeconds: 60\n")
                out.extend(new_block)
                i = block_end
                bumped = True
                continue
            i += 1
        return "".join(out)

    text = _loosen_felix_probe(text, "-felix-live")
    text = _loosen_felix_probe(text, "-felix-ready")

    if mt > 0:
        text, n2 = _sub_once(
            text,
            r"^(\s+veth_mtu: )\"0\"$",
            rf'\1"{mt}"',
            flags=re.MULTILINE,
        )
        if n2 != 1:
            print("WARN: veth_mtu line not updated (mtu>0 but pattern not found once)", file=sys.stderr)

    open(out_path, "w", encoding="utf-8").write(text)


if __name__ == "__main__":
    main()
