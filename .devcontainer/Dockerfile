FROM mcr.microsoft.com/devcontainers/base:trixie
LABEL maintainer "mikoto2000 <mikoto2000@gmail.com>"

ENV PATH=$PATH:/usr/local/go/bin

RUN curl -fsSL https://dl.yarnpkg.com/debian/pubkey.gpg \
  | gpg --dearmor > /usr/share/keyrings/yarn-archive-keyring.gpg \
  && chmod a+r /usr/share/keyrings/yarn-archive-keyring.gpg

RUN curl -LO https://go.dev/dl/go1.26.0.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz \
    && rm go1.26.0.linux-amd64.tar.gz

