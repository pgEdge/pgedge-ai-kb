# RPM spec for pgedge-postgres-mcp-kb.
#
# Sources are assembled into ~/rpmbuild/SOURCES/ by build-rpm.sh:
#   - pgedge-ai-kb-builder_<ver>_linux_<arch>.tar.gz (goreleaser binary)
#   - kb.db (downloaded by validate-kb-db, copied from container mount)
#   - pgedge-ai-kb-builder.yaml (example config)
#   - VERSION (rendered metadata)
#
# Macros %%{pgedge_kb_version}, %%{pgedge_kb_buildnum}, %%{builder_version},
# and %%{arch} are set by build-rpm.sh's rpmbuild --define flags.

%global sname  pgedge-postgres-mcp-kb

Name:       %{sname}
Version:    %{pgedge_kb_version}
Release:    %{pgedge_kb_buildnum}%{?dist}
Summary:    pgEdge PostgreSQL MCP Knowledge Base
License:    PostgreSQL License
URL:        https://github.com/pgEdge/pgedge-ai-kb

Source0:    pgedge-ai-kb-builder_%{builder_version}_linux_%{arch}.tar.gz
Source1:    kb.db
Source2:    pgedge-ai-kb-builder.yaml

BuildArch:  %{_arch}

%description
The pgEdge PostgreSQL MCP Knowledge Base bundles the kb-builder binary
and a pre-built SQLite knowledgebase of PostgreSQL and pgEdge
documentation. The pgEdge PostgreSQL MCP Server consumes the
knowledgebase for semantic search; the kb-builder binary regenerates
or extends it from configured documentation sources.

# ============================================================================
# Build section
# ============================================================================
%prep
# Extract the goreleaser binary tarball into a known subdir.
mkdir -p %{_builddir}/builder
tar -xzf %{SOURCE0} -C %{_builddir}/builder

%build
# Generate and sign SBOMs for both the binary directory and the kb.db file.
syft dir:%{_builddir}/builder -o cyclonedx-json \
    > %{_builddir}/%{sname}-sbom.json || exit 1
syft file:%{SOURCE1} -o cyclonedx-json \
    > %{_builddir}/%{sname}-db-sbom.json || exit 1

KEY_ID=$(gpg --list-secret-keys --with-colons | awk -F: '/^sec/{print $5}' | head -n 1)
export KEY_ID
gpg --armor --detach-sign --default-key "$KEY_ID" \
    --output %{_builddir}/%{sname}-sbom.json.asc \
    %{_builddir}/%{sname}-sbom.json || exit 1
gpg --armor --detach-sign --default-key "$KEY_ID" \
    --output %{_builddir}/%{sname}-db-sbom.json.asc \
    %{_builddir}/%{sname}-db-sbom.json || exit 1

%install
install -d %{buildroot}%{_bindir}
install -d %{buildroot}%{_sysconfdir}/pgedge
install -d %{buildroot}%{_datadir}/pgedge/postgres-mcp-kb
install -d %{buildroot}%{_datadir}
install -d %{buildroot}%{_defaultdocdir}/%{sname}

# Binary — installed as kb-builder; the goreleaser tarball ships it as
# pgedge-ai-kb-builder, renamed at install time for a friendlier PATH name.
install -m 755 %{_builddir}/builder/pgedge-ai-kb-builder \
    %{buildroot}%{_bindir}/kb-builder

# Knowledge base database
install -m 644 %{SOURCE1} \
    %{buildroot}%{_datadir}/pgedge/postgres-mcp-kb/kb.db

# Example config (preserved across upgrades)
install -m 644 %{SOURCE2} \
    %{buildroot}%{_sysconfdir}/pgedge/pgedge-ai-kb-builder.yaml

# Documentation
install -m 644 %{_builddir}/builder/README.md \
    %{buildroot}%{_defaultdocdir}/%{sname}/README.md
install -m 644 %{_builddir}/builder/LICENSE.md \
    %{buildroot}%{_defaultdocdir}/%{sname}/LICENSE.md

# VERSION metadata — KB_DB_RELEASE_TAG, GITHUB_SHA, REPO_TYPE flow in
# from the env exported by build-rpm.sh and pgedge-builder-action.
cat > %{buildroot}%{_defaultdocdir}/%{sname}/VERSION << EOF
PGEDGE_POSTGRES_MCP_KB_VERSION=%{version}-%{release}
BUILDER_VERSION=%{builder_version}
KB_DB_RELEASE_TAG=${KB_DB_RELEASE_TAG:-unknown}
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_COMMIT=${GITHUB_SHA:-unknown}
REPO_TYPE=${REPO_TYPE:-unknown}
EOF

# SBOMs
install -p -m 644 %{_builddir}/%{sname}-sbom.json \
    %{buildroot}%{_datadir}/%{sname}-sbom.json
install -p -m 644 %{_builddir}/%{sname}-sbom.json.asc \
    %{buildroot}%{_datadir}/%{sname}-sbom.json.asc
install -p -m 644 %{_builddir}/%{sname}-db-sbom.json \
    %{buildroot}%{_datadir}/%{sname}-db-sbom.json
install -p -m 644 %{_builddir}/%{sname}-db-sbom.json.asc \
    %{buildroot}%{_datadir}/%{sname}-db-sbom.json.asc

%files
%license %{_defaultdocdir}/%{sname}/LICENSE.md
%doc %{_defaultdocdir}/%{sname}/README.md
%doc %{_defaultdocdir}/%{sname}/VERSION
%{_bindir}/kb-builder
%config(noreplace) %{_sysconfdir}/pgedge/pgedge-ai-kb-builder.yaml
%{_datadir}/pgedge/postgres-mcp-kb/kb.db
%{_datadir}/%{sname}-sbom.json
%{_datadir}/%{sname}-sbom.json.asc
%{_datadir}/%{sname}-db-sbom.json
%{_datadir}/%{sname}-db-sbom.json.asc

%clean
rm -rf %{buildroot}

%changelog
* Tue May 19 2026 pgEdge Team <support@pgedge.com> - 1.0.0
- Initial standalone release.
- Bundles kb-builder binary and pre-built kb.db SQLite knowledgebase.
- Split out of pgedge-postgres-mcp; previously shipped as a subpackage.
