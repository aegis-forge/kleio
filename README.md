<p align="center">
  <img width="100" src="assets/branding/logo.svg" alt="kleio logo"> <br><br>
  <img src="https://img.shields.io/badge/go-v1.25.7-blue" alt="Go version">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
</p>

# kleio

*Kleio* is a crawler for GitHub workflows' histories. From workflows, it extracts all its GitHub Action, Docker, and reusable workflows dependencies. Thanks to this tool, researchers can analyze the software supply chain of GitHub workflows, and how these change over time.

## How to Run

Before starting any of the procedures below, make sure you have duplicated the `env.template` file. After doing so, add the necessary data and rename the file to `.env`.

> [!IMPORTANT]  
> Please note that you need to change `localhost` to `neo` or `mongo` (inside of `.env`) if you want to run Kleio with Docker

For crawling custom repositories (by default the top `env.SIZE * env.PAGES` repositories are crawled), you need to create a `repositories.txt` file at the root of this repository. The file should be structured as follows (be sure to add a newline at the end of the file):

```
https://github.com/aegis-forge/soteria
https://github.com/aegis-forge/cage
https://github.com/aegis-forge/kleio

```

### Docker

Kleio comes with a pre-made dockerfile and docker compose specification file. Moreover, pre-build Docker images are available both on the [ghcr.io](https://github.com/aegis-forge/kleio/pkgs/container/kleio) and [docker](https://hub.docker.com/repository/docker/aegisforge/kleio/general) registries. In both cases we provide images for both the `amd64` and `arm64` arechitectures. However, you can also manually build it. To do so, use the following command:

```sh
docker build -t kleio .
```

By default, the docker compose's kleio image points to the image hosted on Docker. However, this can be changed to `ghcr.io/aegis-forge/kleio:latest` (if using the GitHub version), or `kleio:latest` (if building the Docker image directly from source). Once set up, the compose file can be started by using the following command:

```sh
docker compose up -d
```

### Locally

Before running Kleio, please make sure that the following requirements are satisfied:

- Golang @v1.23.3
- Python @v3.12.4
- Neo4j @v5.26.9
- MongoDB @v6.0
- Yarn @v1.22.22
- [GAWD (modified)](https://github.com/aegis-forge/gawd) @v1.1.1 [↩](#installing-modified-gawd)

After having installed all the requirements, go ahead and compile and run Kleio by using the following command from the root of this repository:

```bash
go build -o kleio ./cmd
./kleio
```

## Installing Modified GAWD

To locally install our modified version of the [original GAWD tool](https://github.com/pooya-rostami/gawd), execute the following (otherwise use the provided dockerfile):

```bash
pip install git+https://github.com/aegis-forge/gawd
```

## Publications

Kleio was used in the following research papers:

- Riggio, E. and Pautasso C. (2026). Changing Nothing, Yet Changing Everything: Exploring Rug Pulls in GitHub Workflows. Proceedings of the 23rd IEEE International Conference on Software Architecture (ICSA), IEEE, in press

## Contacts

- Edoardo Riggio - [https://edoriggio.com](https://edoriggio.com)
