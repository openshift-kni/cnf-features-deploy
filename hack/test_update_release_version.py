import re
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SOURCE = Path(__file__).resolve().parent.parent
FILES = (
    ".gitmodules",
    "hack/common.sh",
    "cnf-tests/mirror/images.json",
    "cnf-tests/testsuites/pkg/images/images.go",
    "cnf-tests/.konflux/Dockerfile",
    "cnf-tests/Dockerfile.openshift",
    "ztp/resource-generator/Containerfile",
    "hack/update-release-version.py",
)


class UpdateReleaseVersionTest(unittest.TestCase):
    def test_complete_partial_upgrade(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in FILES:
                target = root / name
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(SOURCE / name, target)
            shutil.copytree(SOURCE / ".tekton", root / ".tekton")

            common = root / "hack/common.sh"
            dockerfile = root / "cnf-tests/.konflux/Dockerfile"
            current = re.search(r"ENV OCP_VERSION=([0-9]+\.[0-9]+)", dockerfile.read_text()).group(1)
            target = "5.1" if current == "5.0" else "5.0"
            oc = "4.22" if target == "5.1" else "4.21"

            common.write_text(re.sub(r"OCP_VERSION:-[0-9]+\.[0-9]+", f"OCP_VERSION:-{target}", common.read_text()))
            dockerfile.write_text(re.sub(
                r"FROM registry\.redhat\.io/openshift[0-9]+/ose-cli-rhel9:v[0-9]+\.[0-9]+ AS oc",
                f"FROM registry.redhat.io/openshift{target.split('.')[0]}/ose-cli-rhel9:v{target} AS oc",
                dockerfile.read_text(),
            ))
            gitmodules = root / ".gitmodules"
            if f"release-{current}" in gitmodules.read_text():
                gitmodules.write_text(gitmodules.read_text().replace(f"release-{current}", f"release-{target}", 1))
            slug = current.replace(".", "-")
            new_slug = target.replace(".", "-")
            pipeline = root / f".tekton/cnf-tests-{slug}-push.yaml"
            pipeline.rename(root / f".tekton/cnf-tests-{new_slug}-push.yaml")

            script = root / "hack/update-release-version.py"
            check = subprocess.run([sys.executable, script, "--check"], capture_output=True, text=True)
            self.assertNotEqual(check.returncode, 0)
            self.assertIn("expected", check.stderr)

            update = subprocess.run([sys.executable, script, "--set", target], capture_output=True, text=True)
            self.assertEqual(update.returncode, 0, update.stderr)

            check = subprocess.run([sys.executable, script, "--check"], capture_output=True, text=True)
            self.assertEqual(check.returncode, 0, check.stderr)
            self.assertIn(f"aligned at {target} (oc {oc},", check.stdout)
            self.assertEqual(len(list((root / ".tekton").glob(f"*-{new_slug}-*.yaml"))), 4)
            self.assertEqual(len(list((root / ".tekton").glob(f"*-{slug}-*.yaml"))), 0)
            self.assertIn(f"openshift{oc.split('.')[0]}/ose-cli-rhel9:v{oc} AS oc", dockerfile.read_text())
            go_builder = "1.26" if target == "5.1" else "1.25"
            self.assertIn(f"rhel_9_golang_{go_builder} AS builder-stresser", dockerfile.read_text())

            common.write_text(common.read_text().replace(f"OCP_VERSION:-{target}", "OCP_VERSION:-4.21"))
            repair = subprocess.run([sys.executable, script, "--set", target], capture_output=True, text=True)
            self.assertEqual(repair.returncode, 0, repair.stderr)
            self.assertIn(f"OCP_VERSION:-{target}", common.read_text())



if __name__ == "__main__":
    unittest.main()
