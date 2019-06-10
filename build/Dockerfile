FROM registry.access.redhat.com/ubi8-minimal

ENV OPERATOR=/usr/local/bin/teiid-operator

# install operator binary
COPY build/_output/bin/teiid-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER 1001
