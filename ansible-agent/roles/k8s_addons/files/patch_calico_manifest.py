#!/usr/bin/env python3
"""Patch upstream calico.yaml: IPv4 pool CIDR, VXLAN vs bird, optional veth MTU in calico-config."""
import re
import sys


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
    text, n = re.subn(
        r"^\s*#\s*-\s*name:\s*CALICO_IPV4POOL_CIDR\n\s*#\s*value:\s*\"192.168.0.0/16\"",
        f"            - name: CALICO_IPV4POOL_CIDR\n              value: \"{cidr}\"",
        text,
        count=1,
        flags=re.MULTILINE,
    )
    if n != 1:
        print("ERROR: could not find commented CALICO_IPV4POOL_CIDR block to uncomment", file=sys.stderr)
        sys.exit(2)
    if mt > 0:
        text, n2 = re.subn(
            r"^(\s+veth_mtu: )\"0\"$",
            rf'\1"{mt}"',
            text,
            count=1,
            flags=re.MULTILINE,
        )
        if n2 != 1:
            print("WARN: veth_mtu line not updated (mtu>0 but pattern not found once)", file=sys.stderr)
    open(out_path, "w", encoding="utf-8").write(text)


if __name__ == "__main__":
    main()
