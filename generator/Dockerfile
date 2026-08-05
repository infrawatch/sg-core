FROM fedora:32
ENV D=/Generator

WORKDIR $D
COPY . $D/

RUN dnf install -y gcc make qpid-proton-c-devel
RUN make && mv gen /gen

RUN useradd -ms /sbin/nologin sg
USER sg

ENTRYPOINT ["/gen"]
