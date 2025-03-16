# gcp-iam-fuzz

**`gcp-iam-fuzz`** is a tool to quickly enumerate IAM permissions for a Google Cloud Platform (GCP) account using the `testIamPermissions` feature. This is especially useful for when the account cannot directly fetch IAM permissions.

## How it Works

Internally, this tool works by querying the `testIamPermissions` endpoint with the maximum number of permissions to test (100 per request). If any permissions are granted under the current context, the endpoint will return an array of the granted permissions. This gives us the ability to enumerate all possible permissions with only about 100 HTTP requests.

## Usage

```
Usage:
  gcp-iam-fuzz [flags]

Flags:
  -d, --debug            Enable debug logging
  -h, --help             help for gcp-iam-fuzz
  -j, --json             Enable JSON output
  -l, --log-json         Log messages in JSON format
  -o, --output string    Output file path
  -p, --project string   GCP project name
  -T, --threads int      Number of concurrent threads (default 6)
  -t, --token string     GCP access token. environment variable GCP_ACCESS_TOKEN may also be used
```

To use `gcp-iam-fuzz`, you first need an access token. You can use one from an authenticated [`gcloud`](https://cloud.google.com/sdk/docs/install#linux) session on Linux like so:

```shell
# Get access token
export GCP_ACCESS_TOKEN=$(gcloud auth print-access-token)

# Fuzz IAM permissions
./gcp-iam-fuzz -p "replace-with-project-id"
```

> [!WARNING]
> For security purposes, it is recommended to provide the access token via the GCP_ACCESS_TOKEN environment variable.

## Disclaimer

This tool is designed and intended for responsible use in authorized environments. If you need some cloud hacking labs, check out [PwnedLabs](https://pwnedlabs.io/) :)

## Inspiration

This tool was heavily inspired by [hac01](https://github.com/hac01)'s Python script for the same purpose: [gcp-iam-brute](https://github.com/hac01/gcp-iam-brute)
