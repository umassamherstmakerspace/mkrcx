#!/usr/bin/env python3
"""Prepare or apply an exact-image staging rollout on Spence. Production is read-only."""
import argparse
import json
import os
import re
import subprocess
import tempfile
from pathlib import Path

KUBECTL = ["sudo", "-n", "k3s", "kubectl", "-n", "default"]


def kube(*args):
    return subprocess.check_output(KUBECTL + list(args), text=True)


def get(kind, name):
    return json.loads(kube("get", kind, name, "-o", "json"))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--frontend", required=True)
    parser.add_argument("--backend", required=True)
    parser.add_argument("--dedicated-collector", action="store_true", help="Use only after the separate staging collector is verified")
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    for component, image in (("frontend", args.frontend), ("leash", args.backend)):
        if not re.fullmatch(r"ghcr.io/umassamherstmakerspace/mkrcx-" + component + r"@sha256:[a-f0-9]{64}", image):
            raise RuntimeError("An immutable, verified mkr.cx image is required")
    if get("configmap", "mkrcx-leash-staging-config")["data"]["DB_TABLE"] != "mkrcx-staging":
        raise RuntimeError("Staging database isolation check failed")
    # Existence/key check only. Secret values are never logged or copied into a patch.
    if "token" not in get("secret", "mkrcx-printer-fleet-staging-ingest").get("data", {}):
        raise RuntimeError("Staging collector secret is missing")
    production = {n: get("deployment", n)["spec"] for n in ("mkrcx-frontend", "mkrcx-leash")}
    plans = []
    for name, image in (("mkrcx-leash-staging", args.backend), ("mkrcx-frontend-staging", args.frontend)):
        current = get("deployment", name)
        containers = current["spec"]["template"]["spec"]["containers"]
        target_name = "mkrcx-leash" if "leash" in name else "mkrcx-frontend"
        index = next(i for i, c in enumerate(containers) if c["name"] == target_name)
        old = containers[index]
        updated = json.loads(json.dumps(old))
        updated["image"] = image
        environment = updated.setdefault("env", [])
        if "leash" in name:
            variable = {"name": "PRINTER_FLEET_INGEST_SECRET", "valueFrom": {"secretKeyRef": {"name": "mkrcx-printer-fleet-staging-ingest", "key": "token"}}}
        else:
            variable = {"name": "PRINTER_REGISTRY_DEDICATED_COLLECTOR", "value": str(args.dedicated_collector).lower()}
        environment[:] = [e for e in environment if e["name"] != variable["name"]] + [variable]
        path = f"/spec/template/spec/containers/{index}"
        forward = [{"op": "test", "path": "/metadata/resourceVersion", "value": current["metadata"]["resourceVersion"]}, {"op": "replace", "path": path, "value": updated}]
        reverse = [{"op": "test", "path": path + "/image", "value": image}, {"op": "replace", "path": path, "value": old}]
        plans.append((name, forward, reverse))
    os.umask(0o077)
    directory = Path(tempfile.mkdtemp(prefix="mkrcx-printer-registry-staging-", dir="/tmp"))
    for name, forward, reverse in plans:
        (directory / (name + "-forward.json")).write_text(json.dumps(forward))
        (directory / (name + "-rollback.json")).write_text(json.dumps(reverse))
    print("Prepared staging-only patches and rollback in " + str(directory))
    if args.apply:
        for name, _, _ in plans:
            kube("patch", "deployment", name, "--type=json", "--patch-file=" + str(directory / (name + "-forward.json")))
            subprocess.run(KUBECTL + ["rollout", "status", "deployment/" + name, "--timeout=180s"], check=True)
        for name, spec in production.items():
            if get("deployment", name)["spec"] != spec:
                raise RuntimeError("Production spec changed during the staging rollout; investigate before continuing")
        print("Staging rollout ready; production deployment specs unchanged")


if __name__ == "__main__":
    main()
