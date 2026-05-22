ARG baseimage=rippleci/rippled
# Pinned to the latest stable release tag so the cross-impl simulators are
# reproducible. `develop` rebuilds frequently and shifts cross-impl behavior
# (e.g. consensus / propagation regressed silently on 2026-05-21 with the
# 3.2.0-b0 develop image rebuilt during a rxrpl session). Bump deliberately
# when a new stable tag is released and re-validate the hive suite.
ARG tag=3.1.3

FROM --platform=linux/amd64 $baseimage:$tag

# rippleci images run as a non-root user; the build steps below need root
# to install packages and write to /.
USER root

# Install python3 for validators.json conversion. rippleci images have
# been both RHEL- and Debian-based across versions, so try both.
RUN (apt-get update && apt-get install -y --no-install-recommends python3 && rm -rf /var/lib/apt/lists/*) \
    || (dnf install -y python3 && dnf clean all) \
    || true

# Write version.
RUN rippled --version > /version.txt 2>&1 || echo "rippled-unknown" > /version.txt

# Create config and data directories.
RUN mkdir -p /etc/rippled /var/lib/rippled/db /hive-bin

# Copy the known amendment list for this version.
# When updating to a new rippled version, regenerate this file from the
# client's Amendments:DBG log output.
ADD all_amendments.txt /etc/rippled/all_amendments.txt

# Add entry script.
ADD xrpl_start.sh /xrpl_start.sh
RUN chmod +x /xrpl_start.sh

EXPOSE 5005 6006 51235

ENTRYPOINT ["/xrpl_start.sh"]
