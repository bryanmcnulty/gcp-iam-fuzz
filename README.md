# gcp-iam-fuzz
Tool to quickly enumerate IAM permissions for a Google Cloud Platform (GCP) account

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
  -T, --threads int      Number of concurrent threads (default 10)
  -t, --token string     GCP access token
```

To use `gcp-iam-fuzz`, you first need an access token. You can get one from an authenticated [`gcloud` CLI](https://cloud.google.com/sdk/docs/install#linux) session like so:
```bash
gcloud auth print-access-token
```
