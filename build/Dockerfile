FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/teiid-operator

# install operator binary
COPY build/_output/bin/teiid-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# Add conf directory
COPY build/conf /conf

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER 1001
