<p align="center">
  <img width="100" src="assets/branding/logo.svg" alt="kleio logo"> <br><br>
  <img src="https://img.shields.io/badge/go-v1.23.3-blue" alt="Go version">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
</p>

# kleio

## How to Run

Before starting any of the procedures below, make sure you have duplicated both the `env.template` file. After doing so, add the necessary data and rename the file to `.env`.

> [!IMPORTANT]  
> Please note that you need to change `localhost` to `neo` or `mongo` (inside of `.env`) if you want to run Kleio with Docker

### Docker

Kleio comes with a pre-made dockerfile and docker compose specification file. To use these, run the following commands:

```bash
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
go -C ./app build -o ../kleio && ./kleio # Compile and run Kleio
```

## Installing Modified GAWD

To locally install our modified version of the [original GAWD tool](https://github.com/pooya-rostami/gawd), execute the following (otherwise use the provided dockerfile):

```bash
pip install git+https://github.com/aegis-forge/gawd
```
