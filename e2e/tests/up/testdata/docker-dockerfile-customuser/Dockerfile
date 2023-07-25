FROM golang:1.19.11-bullseye

ARG UNAME=vscode
ARG UID=1007
ARG GID=1007

RUN groupadd -g $GID -o $UNAME
RUN useradd -m -u $UID -g $GID -o -s /bin/bash $UNAME

USER $UNAME

CMD /bin/bash