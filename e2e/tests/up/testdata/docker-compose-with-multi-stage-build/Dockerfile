FROM mcr.microsoft.com/vscode/devcontainers/ruby:3.2-bullseye AS base

RUN apt update
RUN apt install vim -y

FROM base

RUN mkdir -p /app
WORKDIR /app