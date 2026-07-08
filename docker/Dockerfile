# Minimal OCI image carrying a single kb.db, for mounting via Kubernetes
# Image Volume Source. One Dockerfile for every provider/model: the DB
# filename is passed as the DB build arg, so no per-provider Dockerfile is
# needed. Build context must contain the named .db file.
#
#   docker build --build-arg DB=kb-<provider>-<model>.db \
#       -f docker/Dockerfile <context-with-the-db>
FROM scratch
ARG DB
COPY ${DB} /${DB}
