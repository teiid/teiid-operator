FROM registry.access.redhat.com/ubi8/openjdk-11:1.3

ENV OPERATOR=/usr/local/bin/teiid-operator

# install operator binary
COPY build/_output/bin/teiid-operator ${OPERATOR}

COPY build/bin /usr/local/bin

USER 0
RUN  /usr/local/bin/user_setup

# Add conf directory
COPY build/conf /conf

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER 1001
