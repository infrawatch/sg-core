# run_retrodep.sh
#
# This script is used to keep status of the various licenses used by the vendored imports.
# Resulting files are used for packaging, and generally will go away in the future once
# packaging systems can account for this.
#
# This script should be run from the root repository pass whenever `go mod vendor` is executed.
#
# Requires the following components:
#  * https://github.com/google/licenseclassifier
#  * https://github.com/grs/go-license-summary
#  * https://github.com/release-engineering/retrodep

retrodep -importpath github.com/infrawatch/sg-core . | tee ./rh-manifest.txt
./build/update_license_info ./rh-manifest.txt .
