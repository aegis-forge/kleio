FROM golang:1.23.3-bookworm

WORKDIR /kleio

COPY . .

# Install the required linux dependencies
RUN apt-get update && apt-get install -y python3-pip python3-venv

# Download modified GAWD package
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install git+https://github.com/aegis-forge/gawd

# Download dependencies and build Kleio
RUN go mod download
RUN go -C ./cmd build -o ../kleio

# Remove unnecessary directories and files
RUN find . -maxdepth 1 | grep -v "kleio\|\.ini\|repositories\|^.$" | xargs rm -rf

CMD [ "/kleio/kleio" ]
