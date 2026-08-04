FROM fedora:32
ENV D=/Generator

WORKDIR $D
COPY . $D/

RUN dnf install -y gcc make qpid-proton-c-devel
RUN make && mv gen /gen

RUN echo "sg:x:1001:0:sg user:/:/sbin/nologin" >> /etc/passwd
USER 1001

ENTRYPOINT ["/gen"]
