#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
STAGE_DIR="${DIST_DIR}/deb-stage"
PKG_NAME="awesomer"
VERSION="${1:-0.0.0}"
ARCH="${2:-$(dpkg --print-architecture 2>/dev/null || echo amd64)}"
OUTPUT_DEB="${DIST_DIR}/${PKG_NAME}_${VERSION}_${ARCH}.deb"
GO_BIN="${GO_BIN:-go}"

rm -rf "${STAGE_DIR}" "${OUTPUT_DEB}"
mkdir -p \
  "${STAGE_DIR}/DEBIAN" \
  "${STAGE_DIR}/usr/bin" \
  "${STAGE_DIR}/etc/awesomer" \
  "${STAGE_DIR}/lib/systemd/system"

cat > "${STAGE_DIR}/DEBIAN/control" <<EOF
Package: ${PKG_NAME}
Version: ${VERSION}
Section: admin
Priority: optional
Architecture: ${ARCH}
Maintainer: Awesomer Maintainers
Description: Awesomer process monitor client and background daemon
EOF

cat > "${STAGE_DIR}/DEBIAN/postinst" <<'EOF'
#!/usr/bin/env bash
set -e
systemctl daemon-reload >/dev/null 2>&1 || true
EOF
chmod 0755 "${STAGE_DIR}/DEBIAN/postinst"

cat > "${STAGE_DIR}/DEBIAN/prerm" <<'EOF'
#!/usr/bin/env bash
set -e
if [ "${1:-}" = "remove" ]; then
  systemctl stop awesomerd.service >/dev/null 2>&1 || true
  systemctl disable awesomerd.service >/dev/null 2>&1 || true
fi
EOF
chmod 0755 "${STAGE_DIR}/DEBIAN/prerm"

cat > "${STAGE_DIR}/DEBIAN/conffiles" <<'EOF'
/etc/awesomer/config.yaml
EOF

(
  cd "${ROOT_DIR}"
  "${GO_BIN}" build -o "${STAGE_DIR}/usr/bin/awesomerctl" ./cmd/awesomerctl
  "${GO_BIN}" build -o "${STAGE_DIR}/usr/bin/awesomerd" ./cmd/awesomerd
)

install -m 0644 "${ROOT_DIR}/deploy/systemd/awesomerd.service" "${STAGE_DIR}/lib/systemd/system/awesomerd.service"
install -m 0644 "${ROOT_DIR}/internal/config/config.yaml.example" "${STAGE_DIR}/etc/awesomer/config.yaml"

dpkg-deb --build --root-owner-group "${STAGE_DIR}" "${OUTPUT_DEB}"
echo "Built ${OUTPUT_DEB}"
