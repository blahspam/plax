################################################
# Build
FROM golang:1.26-alpine AS build

COPY ./ /src
WORKDIR /src
RUN go build ./plax.go

################################################
# Package
FROM alpine:3.23
COPY --from=build /src/plax /opt/plax

ARG CREATED=unknown
ARG URL=https://github.com/blahspam/plax
ARG VERSION=unknown
ARG REVISION=unknown
ARG PUID=1000
ARG PGID=1000

USER ${PUID}:${PGID}

LABEL \
    org.opencontainers.image.created=${CREATED} \
    org.opencontainers.image.authors="jeff@blahspam.com" \
    org.opencontainers.image.url=${URL} \
    org.opencontainers.image.documentation=${URL} \
    org.opencontainers.image.source=${URL} \
    org.opencontainers.image.version=${VERSION} \
    org.opencontainers.image.revision=${REVISION} \
    org.opencontainers.image.licenses=MIT \
    org.opencontainers.image.title="Plex Local Asset Exporter" \
    org.opencontainers.image.description="Export Plex posters and backgrounds to the corresponding show, season, or movie directory"

ENV \
  PLEX_URL=http://127.0.0.1:32400 \
  PLEX_TOKEN=my_token \
  LIBRARY=all

CMD /opt/plax -u ${PLEX_URL} -t ${PLEX_TOKEN} -l "${LIBRARY}"
