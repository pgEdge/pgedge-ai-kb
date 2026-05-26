# RPM spec for pgedge-ai-kb.
#
# Sources are assembled into ~/rpmbuild/SOURCES/ by build-rpm.sh:
#   - kb.db (downloaded by validate-kb-db, copied from container mount)
#   - LICENSE.md, README.md (from this repo's root)
#
# This is a data-only, architecture-independent package. The kb-builder
# binary and its example config are NOT shipped here — they're distributed
# via goreleaser tarballs on GitHub Releases and bundled inside the
# pgedge-postgres-mcp server package.
#
# Macros %%{pgedge_kb_version} and %%{pgedge_kb_buildnum} are set by
# build-rpm.sh's rpmbuild --define flags.

%global sname  pgedge-ai-kb

Name:       %{sname}
Version:    %{pgedge_kb_version}
Release:    %{pgedge_kb_buildnum}%{?dist}
Summary:    pgEdge AI Knowledgebase database
License:    PostgreSQL License
URL:        https://github.com/pgEdge/pgedge-ai-kb

Source0:    kb.db
Source1:    LICENSE.md
Source2:    README.md

BuildArch:  noarch

%description
Pre-built SQLite knowledgebase of PostgreSQL and pgEdge documentation
with embeddings from one or more providers (OpenAI, Voyage, Ollama).
Consumed by the pgEdge PostgreSQL MCP Server and the pgEdge AI DBA
Workbench for semantic search. Regenerate or extend it with the
pgedge-ai-kb-builder binary distributed separately via GitHub Releases.

# ============================================================================
# Build section
# ============================================================================
%prep
# Nothing to extract — sources are copied directly in %install.

%build
# Generate and sign an SBOM for the kb.db file.
syft file:%{SOURCE0} -o cyclonedx-json \
    > %{_builddir}/%{sname}-sbom.json || exit 1

KEY_ID=$(gpg --list-secret-keys --with-colons | awk -F: '/^sec/{print $5}' | head -n 1)
export KEY_ID
gpg --armor --detach-sign --default-key "$KEY_ID" \
    --output %{_builddir}/%{sname}-sbom.json.asc \
    %{_builddir}/%{sname}-sbom.json || exit 1

%install
install -d %{buildroot}%{_datadir}/pgedge/postgres-mcp-kb
install -d %{buildroot}%{_datadir}
install -d %{buildroot}%{_defaultdocdir}/%{sname}

# Knowledge base database — path is stable so the pgedge-postgres-mcp
# server's default config keeps working after the package rename.
install -m 644 %{SOURCE0} \
    %{buildroot}%{_datadir}/pgedge/postgres-mcp-kb/kb.db

# Documentation
install -m 644 %{SOURCE1} \
    %{buildroot}%{_defaultdocdir}/%{sname}/LICENSE.md
install -m 644 %{SOURCE2} \
    %{buildroot}%{_defaultdocdir}/%{sname}/README.md

# VERSION metadata — KB_DB_RELEASE_TAG, GITHUB_SHA, REPO_TYPE flow in
# from the env exported by build-rpm.sh and pgedge-builder-action.
cat > %{buildroot}%{_defaultdocdir}/%{sname}/VERSION << EOF
PGEDGE_AI_KB_VERSION=%{version}-%{release}
KB_DB_RELEASE_TAG=${KB_DB_RELEASE_TAG:-unknown}
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_COMMIT=${GITHUB_SHA:-unknown}
REPO_TYPE=${REPO_TYPE:-unknown}
EOF

# SBOM
install -p -m 644 %{_builddir}/%{sname}-sbom.json \
    %{buildroot}%{_datadir}/%{sname}-sbom.json
install -p -m 644 %{_builddir}/%{sname}-sbom.json.asc \
    %{buildroot}%{_datadir}/%{sname}-sbom.json.asc

%files
%license %{_defaultdocdir}/%{sname}/LICENSE.md
%doc %{_defaultdocdir}/%{sname}/README.md
%doc %{_defaultdocdir}/%{sname}/VERSION
%{_datadir}/pgedge/postgres-mcp-kb/kb.db
%{_datadir}/%{sname}-sbom.json
%{_datadir}/%{sname}-sbom.json.asc

%clean
rm -rf %{buildroot}

%changelog
* Tue May 26 2026 pgEdge Team <support@pgedge.com> - 1.0.0
- Renamed package from pgedge-postgres-mcp-kb to pgedge-ai-kb.
- Reduced scope to kb.db only (noarch). Binary and example config no
  longer shipped here; binary lives in pgedge-postgres-mcp server pkg
  and as standalone GoReleaser tarballs on GitHub Releases.
* Tue May 19 2026 pgEdge Team <support@pgedge.com> - 1.0.0
- Initial standalone release.
- Bundles kb-builder binary and pre-built kb.db SQLite knowledgebase.
- Split out of pgedge-postgres-mcp; previously shipped as a subpackage.
