FROM golang:1.25.7-alpine3.23 AS build

WORKDIR /kleio

COPY . .

# Install the required linux dependencies
RUN apk update && apk add --no-cache py3-pip py3-virtualenv nodejs npm git

# Install yarn for yarn.lock analysis
RUN npm install --global yarn corepack
RUN corepack enable

# Download modified GAWD package
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install git+https://github.com/aegis-forge/gawd

# Download golang dependencies and build Kleio
RUN go mod download
RUN go -C ./cmd build -o ../kleio


FROM alpine:3.23

WORKDIR /kleio

RUN apk add --no-cache git

# Copy necessary files from build stage
COPY --from=build /kleio/kleio /kleio/kleio
COPY --from=build /kleio/repositories.txt /kleio/repositories.txt

ENTRYPOINT [ "/kleio/kleio" ]
