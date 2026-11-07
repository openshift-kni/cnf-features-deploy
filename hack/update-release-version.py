#!/usr/bin/env python3
"""Update release references together, including partially upgraded files."""

import argparse
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
NUMBER = r"[0-9]+\.[0-9]+"
SLUG = r"[0-9]+-[0-9]+"
GO_BUILDER_BY_RELEASE = {"5.0": "1.25", "5.1": "1.26"}
OC_IMAGE_BY_RELEASE = {"5.0": "4.21", "5.1": "4.22"}

# (expression, expected matches, value kind). These select release fields only;
# dependency versions, addresses, image digests, and Go modules are independent.
FIELDS = {
    "hack/common.sh": (
        (rf'(?m)^export OCP_VERSION="\$\{{OCP_VERSION:-(?P<value>{NUMBER})\}}"$', 1, "release"),
        (rf'(?m)^export OPERATOR_VERSION="\$\{{OPERATOR_VERSION:-(?P<value>{NUMBER})\}}"$', 1, "release"),
    ),
    "cnf-tests/mirror/images.json": (
        (rf'"image": "cnf-tests:(?P<value>{NUMBER})"', 1, "release"),
        (rf'"image": "dpdk:(?P<value>{NUMBER})"', 1, "release"),
    ),
    "cnf-tests/testsuites/pkg/images/images.go": (
        (rf'cnfTestsImage = "cnf-tests:(?P<value>{NUMBER})"', 1, "release"),
        (rf'dpdkTestImage = "dpdk:(?P<value>{NUMBER})"', 1, "release"),
    ),
    "cnf-tests/.konflux/Dockerfile": (
        (rf'(?m)^FROM registry\.redhat\.io/openshift(?P<namespace>[0-9]+)/ose-cli-rhel9:v(?P<value>{NUMBER}) AS oc$', 1, "oc"),
        (rf'openshift-golang-builder:rhel_9_golang_(?P<value>{NUMBER}) AS ', 5, "go_builder"),
        (rf"(?m)^RUN sed -i 's/cnf-tests:(?P<value>{NUMBER})/cnf-tests-rhel9:v(?P<other>{NUMBER})/g' ", 1, "release"),
        (rf"(?m)^RUN sed -i 's/dpdk:(?P<value>{NUMBER})/dpdk-base-rhel9:v(?P<other>{NUMBER})/g' ", 1, "release"),
        (rf'(?m)^ENV OCP_VERSION=(?P<value>{NUMBER})$', 1, "release"),
        (rf'cpe="cpe:/a:redhat:openshift:(?P<value>{NUMBER})::el9"', 1, "release"),
    ),
    "cnf-tests/Dockerfile.openshift": (
        (rf'builder:rhel-9-golang-(?P<value>{NUMBER})-openshift-', 6, "go_builder"),
        (rf'openshift-(?P<value>{NUMBER}) as builder-', 2, "release"),
        (rf'openshift-(?P<value>{NUMBER}) AS builder-', 3, "release"),
        (rf'openshift-(?P<value>{NUMBER}) AS go-builder', 1, "release"),
        (rf'(?m)^FROM registry\.ci\.openshift\.org/ocp/(?P<value>{NUMBER}):(oc-rpms|base-rhel9)', 2, "release"),
        (rf'(?m)^ENV OCP_VERSION=(?P<value>{NUMBER})$', 1, "release"),
    ),
    "ztp/resource-generator/Containerfile": (
        (rf'cpe="cpe:/a:redhat:openshift:(?P<value>{NUMBER})::el9"', 1, "release"),
    ),
    ".gitmodules": (
        (rf'(?m)^\s*branch = release-(?P<value>{NUMBER})$', (0, 3), "release"),
    ),
}


def parse_args():
    parser = argparse.ArgumentParser(description=__doc__)
    action = parser.add_mutually_exclusive_group(required=True)
    action.add_argument("--set", metavar="MAJOR.MINOR")
    action.add_argument("--check", action="store_true")
    parser.add_argument("--go-builder-version", metavar="MAJOR.MINOR")
    parser.add_argument("--oc-image-version", metavar="MAJOR.MINOR")
    args = parser.parse_args()
    for value in (args.set, args.go_builder_version, args.oc_image_version):
        if value and not re.fullmatch(NUMBER, value):
            parser.error("versions must have the form MAJOR.MINOR, for example 5.1")
    return args


def main() -> int:
    args = parse_args()
    current = re.search(rf'(?m)^ENV OCP_VERSION=(?P<value>{NUMBER})$', (ROOT / "cnf-tests/.konflux/Dockerfile").read_text())
    if not current:
        print("Cannot read the current release version from the CNF container Dockerfile", file=sys.stderr)
        return 1
    release = args.set or current.group("value")
    oc = args.oc_image_version or OC_IMAGE_BY_RELEASE.get(release)
    go_builder = args.go_builder_version or GO_BUILDER_BY_RELEASE.get(release)
    if not oc:
        print(f"No known oc image version for {release}; supply OC_IMAGE_VERSION=MAJOR.MINOR", file=sys.stderr)
        return 1
    if not go_builder:
        print(f"No known CNF builder version for {release}; supply GO_BUILDER_VERSION=MAJOR.MINOR", file=sys.stderr)
        return 1
    if args.set and release.split(".")[0] != "5":
        print("This release layout only supports 5.x; other major releases require manual image registry changes", file=sys.stderr)
        return 1

    values = []
    changes = {}
    errors = []
    moves = []
    for name, fields in FIELDS.items():
        path = ROOT / name
        original = path.read_text()
        updated = original
        for pattern, expected, kind in fields:
            matches = list(re.finditer(pattern, original))
            allowed = expected if isinstance(expected, tuple) else (expected,)
            if len(matches) not in allowed:
                errors.append(f"{name}: expected {expected} matches for {pattern!r}, found {len(matches)}")
                continue
            target = {"oc": oc, "go_builder": go_builder}.get(kind, release)
            for match in matches:
                values.append((name, kind, match.group("value")))
                if kind == "oc":
                    values.append((name, "oc_namespace", match.group("namespace")))
                if "other" in match.groupdict():
                    values.append((name, kind, match.group("other")))
            if args.set:
                if kind == "oc":
                    image = f"FROM registry.redhat.io/openshift{oc.split('.')[0]}/ose-cli-rhel9:v{oc} AS oc"
                    updated = re.sub(pattern, lambda match: image, updated)
                else:
                    updated = re.sub(pattern, lambda match: re.sub(NUMBER, target, match.group()), updated)
        changes[path] = updated

    pipelines = {}
    for path in (ROOT / ".tekton").glob("*.yaml"):
        found = re.fullmatch(rf"(cnf-tests|ztp-site-generate)-({SLUG})-(push|pull-request)\.yaml", path.name)
        if not found:
            continue
        component, filename_slug, event = found.groups()
        key = (component, event)
        if key in pipelines:
            errors.append(f"Duplicate Tekton pipeline for {component} {event}")
            continue
        pipelines[key] = path
        original = path.read_text()
        slug_pattern = rf"{re.escape(component)}-(?P<value>{SLUG})"
        pipeline_fields = [
            (rf'target_branch == "(?:master|release-(?P<value>{NUMBER}))"', 1, "release"),
            (slug_pattern, 6, "slug"),
        ]
        if component == "ztp-site-generate":
            pipeline_fields.extend((
                (rf'version=v(?P<value>{NUMBER})-', 1, "release"),
                (rf'(?m)^\s*- release=(?P<value>{NUMBER})$', 1, "release"),
            ))
        updated = original
        values.append((str(path.relative_to(ROOT)), "slug", filename_slug))
        for pattern, expected, kind in pipeline_fields:
            matches = list(re.finditer(pattern, original))
            if len(matches) != expected:
                errors.append(f"{path.relative_to(ROOT)}: expected {expected} matches for {pattern!r}, found {len(matches)}")
                continue
            values.extend((str(path.relative_to(ROOT)), kind, match.group("value"))
                          for match in matches if match.group("value") is not None)
            if args.set:
                replacement = release.replace(".", "-") if kind == "slug" else release
                number = SLUG if kind == "slug" else NUMBER
                updated = re.sub(pattern, lambda match: re.sub(number, replacement, match.group()), updated)
        destination = path.with_name(f"{component}-{release.replace('.', '-')}-{event}.yaml") if args.set else path
        changes[destination] = updated
        if destination != path:
            changes.pop(path, None)
            moves.append((path, destination))
            if destination.exists():
                errors.append(f"Destination already exists: {destination.relative_to(ROOT)}")

    for component in ("cnf-tests", "ztp-site-generate"):
        for event in ("push", "pull-request"):
            if (component, event) not in pipelines:
                errors.append(f"Missing Tekton pipeline for {component} {event}")

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1

    targets = {"release": release, "oc": oc, "oc_namespace": oc.split(".")[0],
               "go_builder": go_builder, "slug": release.replace(".", "-")}
    mismatches = [(name, value, targets[kind]) for name, kind, value in values if value != targets[kind]]
    if args.check:
        for name, value, target in mismatches:
            print(f"{name}: found {value}, expected {target}", file=sys.stderr)
        if mismatches:
            return 1
        print(f"Release references are aligned at {release} (oc {oc}, CNF builder Go {go_builder})")
        return 0

    for source, destination in moves:
        source.rename(destination)
        print(f"Renamed {source.relative_to(ROOT)} to {destination.relative_to(ROOT)}")
    for path, content in changes.items():
        if not path.exists() or path.read_text() != content:
            path.write_text(content)
            print(f"Updated {path.relative_to(ROOT)}")
    print(f"Release references set to {release} (oc {oc}, CNF builder Go {go_builder})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
