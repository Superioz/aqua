# aqua

<img src="./.github/assets/aqua_banner.png" height="160" />

---

**aqua** is a simple file uploading and sharing server for personal use. It is built to be easy to set up and host on your own server, for example to use it in combination with uploading tools like [ShareX](https://getsharex.com/).

It is the successor/rework of my previous project [lightf](https://github.com/Superioz/lightf), but this time without trying weird things out and building a complete product instead.

![aqua_structure](./.github/assets/aqua_structure.png)

The diagram above shows the general structure of the system. It is heavily based on abstracting interfaces for things like the file system or databases. There is also the *Storage Handler* which does all the management work in combining the different data sources (i.e. the physical files and their metadata) together. Every X (configurable!) minutes, the *Cleanup Scheduler* will command the Storage Handler to delete all expired files. This will also happen when the server starts.

# Installation

There are as always multiple ways to install the server. The recommended way is to use Docker Compose or deploy it to Kubernetes, but we start with the manual way.

Also, if you don't want the server to serve the files themselves, then you can disable it via the environment variable described in section [Configuration](#configuration). That could be the case if you already have a Nginx installed and want it to serve the files instead of aqua. That way, you have full customization options.

## Manually

This is only for those people, that are still living in the 90s or are not comfortable with a Docker installation.

1. First download the specific binary from the [releases](https://github.com/Superioz/aqua/releases) and put it somewhere (e.g. with `wget` on Linux). Also, add execution permission with `chmod +x aq-linux-amd64`.
2. Do the configuration you want (e.g. setting environment variables and creating an `auth.yml`). See [Configuration](#configuration)
3. Execute the downloaded binary and don't forget to open port `8765` on your machine/server.

For further instructions like creating a service that can easily be started with `service aqua start`, please refer to other pages (there are a bunch that explain this) - I won't.

## Docker (Compose)

Before following the steps, make sure you have [Docker](https://docs.docker.com/get-docker/) installed. And if you want to use the preconfigured `docker-compose` file, install [Docker Compose](https://docs.docker.com/compose/install/) as well.

Before you can start anything, you need the basis configuration:

1. Download and edit the `auth.yml` with your custom auth tokens and settings.
2. Download and rename the `.env.dist` to `.env` and edit it as well.

For a quick start with Docker, you can use the following command:

```sh
docker run --rm                    \
  --env-file ./.env                \
  -v ./auth.yml:/etc/aqua/auth.yml \
  -v ./files:/var/lib/aqua/        \
  -p 8765:8765                     \
  ghcr.io/superioz/aqua:latest
```

If you are on Windows, you might need to add `MSYS_NO_PATHCONV=1` so that the paths are correctly parsed. Also, to refer to the current directory, use `"$(pwd)"` instead of `./`, because that sometimes makes problems in Windows as well.

For Docker Compose it's easier:

```sh
docker-compose up
```

If you want to build the image yourself instead, you can of course `git clone git@github.com:Superioz/aqua.git` and then execute `docker-compose up --build`.

## Kubernetes

Make sure to have a Kubernetes cluster running somewhere (either on bare-metal or on GCP, AWS or some other provider) and check beforehand if pods inside the cluster can access `ghcr.io`, which is the GitHub Docker registry, which we need for our aqua server image. Of course, you also need `kubectl` to be installed on your system.

1. Clone the repository or copy at least the `/kubernetes` folder into some directory
2. Create a namespace called `aqua` with `kubectl create ns aqua`
3. Apply the configuration files from the `/kubernetes` folder with `kubectl apply -k ./kubernetes/`. The `-k` flag is part of the newer versions of kubectl and allows [Kustomize](https://kustomize.io/) to work.
4. Either create an Ingress or connect to the cluster via `kubectl port-forward svc/aqua-server 8765`
5. Done. Finito. You should now be able to access the server.

# Configuration

## Environment Variables

For examplary usage of the environment variables, have a look at the `.env.dist` file in the root directory of the repository.

| Variable | Description |
| -------- | ----------- |
| `AUTH_CONFIG_PATH` | Path to the `auth.yml` config file. |
| `FILE_STORAGE_PATH` | Path to the directory, where the files should be stored. |
| `FILE_NAME_LENGTH` | Length of the file names, that should be randomly generated. Should be long enough to make guessing impossible. Cannot be longer than 24 characters. |
| `FILE_MAX_SIZE` | Maximum size for uploaded files in Megabytes. |
| `FILE_META_DB_PATH` | Path to the directory, where the sqlite database for file metadata should be stored. Recommended to not be the same folder as `FILE_STORAGE_PATH` to prevent overlapping. |
| `FILE_EXPIRATION_CYCLE` | Determines the interval of the expiration cycle. `5` means that every 5 seconds the files will be checked for expiration.  |
| `FILE_SERVING_ENABLED` | Defaults to `true`, if `false`, the server won't serve the stored files. |
| `FILE_EXTENSIONS_RESPONSE` | Defaults to `true`. if the file name returned will have its extension added to it. |
| `FILE_EXTENSIONS_EXCLUDED` | Comma-seperated list of MIME types, that should be excluded from the extension response rule above. Defaults to `image/png,image/jpeg` |
| `METRICS_ENABLED` | Is normally set to true, but otherwise disables the Prometheus metrics publishing. |

## Tokens

Inside the `auth.yml` file you can configure which tokens are valid and for what file types they can be used for. An example file could look like this:

```yaml
validTokens:
  - token: 71a4c056ab9b0fb965063344cd6616bc
    fileTypes:
      - image/png
      - image/jpeg
```

Each token is an element of the `validTokens` list and can be generated however you want. It is not even necessary for it to be hexadecimal, it can be any kind of string you want (
e.g. `my_favorite_password` or something like `X_!]6rk[wC]%7aN^fB>#PzuFd{L,ugsq`).

But if you want to keep it simple, you can use the `aq` CLI tool to generate it directly like so:

```console
user@host:~$ aq generate --length 32
L1dLUm12!Lb%7Nz1ep4h5Vo+Fn531&EU
```

After adding the token to the list you may want to restrict what files can be uploaded with that token. That can be done with the `fileTypes` field. If you leave it empty, all file types are possible, otherwise only the configured ones.

Normally we would accept every possible MIME type, but as they behave completely different sometimes and we want to keep it simple, we **only support** the following ones:

```
application/gzip
application/json
application/pdf
application/vnd.rar
application/x-7z-compressed
application/zip
audio/mpeg
audio/ogg
audio/opus
audio/webm
image/gif
image/jpeg
image/png
image/svg+xml
text/csv
text/plain
video/mp4
video/mpeg
video/webm
```

# Metrics

We also expose Prometheus metrics to the port `:8766`, if the specific environment variable is not set to `false`. To scrape these metrics simply make sure that they are enabled and that you add them to the Prometheus scrape targets.

If you are in a Kubernetes environment instead, a normal Prometheus deployment should automatically pick up the metrics, because we add specific annotations to the pods. If that does not work, or you are using the Prometheus Operator, you might want to have a look at `ServiceMonitor`s somewhere at the [Prometheus Operator documentation](https://github.com/prometheus-operator/prometheus-operator).

Here is a list of all Prometheus metrics:

| Metric | Description |
| ------ | ----------- |
| aqua_files_uploaded_total | Self explanatory |
| aqua_files_expired_total | Self explanatory lol |

# CLI Tool

Installing the CLI tool is fairly simple. Just download the respective binary from [releases](https://github.com/Superioz/aqua/releases), rename it and put it somewhere in your `$PATH`. Don't forget to add execution permissions `chmod +x aq`.

Now to upload a local file, you just have to execute the following command:

```sh
aq upload --host https://my-domain.com:8765 --token my_token local_file.png
```

To upload multiple files at the same time, just do it like this:

```sh
aq upload --host https://my-domain.com:8765 --token my_token local_file1.png local_file2.txt [...]
```

# ShareX

As mentioned in the first section, you can easily configure ShareX to use aqua as server. For that you can copy the contents below to a file with the `.sxcu` ending like `aqua.sxcu`. Edit it now to your own needs, i.e. replacing the `your-domain.com` with the address of your installation and maybe adjusting the `expiration`, `-1` means that the files won't expire.

After that you can import it to your custom upload goals in the ShareX UI.

```json
{
  "Version": "12.4.0",
  "DestinationType": "ImageUploader, TextUploader",
  "RequestMethod": "POST",
  "RequestURL": "https://your-domain.com/upload",
  "Headers": {
    "Authorization": "Bearer your-generated-token-here"
  },
  "Body": "MultipartFormData",
  "Arguments": {
    "metadata": "{ \"expiration\": 3600 }"
  },
  "FileFormName": "file",
  "URL": "https://your-domain.com/$json:fileName$"
}
```

More information can be found here: [ShareX Custom Uploader Guide](https://getsharex.com/docs/custom-uploader).
