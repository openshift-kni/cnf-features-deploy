# Rendered .yaml for source-crs

## This folder has been moved and contents frozen. Please use [telco-reference](https://github.com/openshift-kni/telco-reference/tree/main/telco-ran).

The mechanisms in this makefile enable us to put plaintext files and systemd
units into source control, generate the corresponding .yaml documents, and
ensure the .yaml documents are kept in-sync with the sources, assuming a
subdirectory follows the following conventions:

- The name of the directory corresponds to the rendered filename, such that a
  directory called `03-sctp-machine-config` will be rendered into
  `../source-crs/extra-manifest/03-sctp-machine-config-{master|worker}.yaml`
- The directory must contain a `build.sh` which is run with $MCPROLE set to the
  expected MCP role to produce.  This script produces to stdout the rendered
  yaml identically every time it is called, provided that the input files
  remain the same (it must not include any date stamps or git commit hashes).
  This is used both to generate the rendered
  `../source-crs/extra-manifest/*.yaml` and to do an integrity check as part of
  the ci-job target which ensures the rendered yaml stays in-sync with the
  source content.
- The directory may contain a `test.sh` which can additionally perform any unit
  test operations on the contents of the directory.

Both `build.sh` and `test.sh` are executed with their working directory set to
their own subdirectory.

##To edit or create a rendered .yaml file:

- Edit or create the appropriate directory and source components, with a
  `build.sh` (and `test.sh` as needed)
- Run `make` to render the `../source-crs/extra-manifest/*.yaml`
- Add the rendered file with the source changes in a single git commit

Github CI 'ci-job' will fail if you don't commit the source changes and the
rendered yaml in the same PR.
