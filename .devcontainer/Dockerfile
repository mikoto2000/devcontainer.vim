FROM mcr.microsoft.com/devcontainers/base:trixie
LABEL maintainer "mikoto2000 <mikoto2000@gmail.com>"

ENV PATH=$PATH:/usr/local/go/bin

RUN curl -fsSL https://dl.yarnpkg.com/debian/pubkey.gpg \
  | gpg --dearmor > /usr/share/keyrings/yarn-archive-keyring.gpg \
  && chmod a+r /usr/share/keyrings/yarn-archive-keyring.gpg

RUN curl -fLO https://go.dev/dl/go1.27.4.linux-amd64.tar.gz \
    && echo '7226736c2339719cda28fbf3a54d71aa63bcefe376e98cef00034268dab24840  go1.27.4.linux-amd64.tar.gz' | sha256sum --check \
    && tar -C /usr/local -xzf go1.27.4.linux-amd64.tar.gz \
    && rm go1.27.4.linux-amd64.tar.gz

