# Janus-go

![E2E GCP](https://github.com/zepellin/janus-go/actions/workflows/e2e-gcp.yaml/badge.svg)
![E2E SA KEY](https://github.com/zepellin/janus-go/actions/workflows/e2e-sa-key.yaml/badge.svg)

## Description

Janus-go is a AWS CLI external source authentication program for use with Google Cloud GKE workload identity or GCE VM identity. It is designed to allow authenticating AWS IAM role from Google Cloud environments (such as GKE cluster or GCE VM instance) without the need of generating long term AWS credentials.

This project was inspired by [Janus](https://github.com/doitintl/janus), a python implementation of the same authentication flow. This project was written in go for easier installation and usage of the program where a single binary is implementation is better suited (such as inside of existing container running on kubernetes).

## Do you need Janus-go?

Most AWS SDKs, and the tools built on them (AWS CLI, AWS Terraform provider, boto3, rclone, ...), can do the Google identity token to AWS credentials exchange themselves. Give the profile a `role_arn` and a `web_identity_token_file` (or set `AWS_ROLE_ARN` and `AWS_WEB_IDENTITY_TOKEN_FILE`), and the SDK calls `AssumeRoleWithWebIdentity` itself, reading the token file again each time it refreshes credentials:

```text
[profile my-aws-account]
role_arn = arn:aws:iam::123456789012:role/my-trusted-role
web_identity_token_file = /var/run/google/token
role_session_name = my-session
```

The SDK only needs a valid Google identity token in that file, requested with the audience the IAM role trust policy expects. Google identity tokens expire after one hour, so something has to keep rewriting the file. On Kubernetes, that is usually a small sidecar container that fetches a token from the metadata server (`http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=...`) and writes it to a volume shared with the application container.

When you can run such a token writer, this approach is more efficient than `credential_process`: all profiles share one token file, no extra process is started for each profile and credential refresh, and the Janus-go binary doesn't have to be in the application container.

On Kubernetes, you can skip Google identities and use the pod's Kubernetes service account token instead. Register the cluster's OIDC issuer as an [IAM OIDC identity provider](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html) in the AWS account, with `sts.amazonaws.com` as its audience. For GKE, the issuer is `https://container.googleapis.com/v1/projects/PROJECT_ID/locations/LOCATION/clusters/CLUSTER_NAME`; for other clusters, its discovery document and signing keys have to be reachable from the internet. Then let the IAM role trust policy allow the Kubernetes service account (condition `<issuer>:sub` equal to `system:serviceaccount:NAMESPACE:SERVICE_ACCOUNT`), and mount a projected service account token for the AWS SDK to read. The kubelet keeps the token fresh, so neither a sidecar nor Janus-go is needed:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app-pod
spec:
  serviceAccountName: my-app
  containers:
    - name: main-container
      ...
      env:
        - name: AWS_ROLE_ARN
          value: arn:aws:iam::123456789012:role/my-trusted-role
        - name: AWS_WEB_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/aws/token
      volumeMounts:
        - mountPath: /var/run/secrets/aws
          name: aws-token
          readOnly: true
  volumes:
    - name: aws-token
      projected:
        sources:
          - serviceAccountToken:
              path: token
              audience: sts.amazonaws.com
              expirationSeconds: 3600
```

Janus-go is still useful when:

- nothing keeps a token file up to date, for example on a GCE VM or in a CI job
- on Kubernetes, the IAM role has to trust a Google identity rather than a Kubernetes service account, and you can't add a token writer sidecar to the pod
- you authenticate outside of Google Cloud with a service account key or `gcloud auth application-default login` user credentials, where there is no metadata server to get a token from
- the tool supports `credential_process` but not `web_identity_token_file`

## Prerequisites

1. The environment in which the program is running has to be able to provide Google Cloud Identity token from Google Cloud metadata server. This can be achieved either by running on GCE VM instance or as a GKE workload with workload identity enabled
2. An AWS IAM role is created with a [trust policy specifying the Google Cloud IAM identity](https://aws.amazon.com/blogs/security/access-aws-using-a-google-cloud-platform-native-workload-identity/) used by VM instance or GKE workload identity from step 1.

## Installation

### Locally

Download appropriate release for your OS and achitecture from the project's release page.

```bash
wget -qO janus-go https://github.com/zepellin/janus-go/releases/download/v0.6.6/janus-v0.6.6-linux-amd64 && chmod +x janus-go
```

### Inside of a Kubernetes pod

To use the binary inside of Kubernetes pod, download the binary using init container and mount the binary path inside of your main container:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app-pod
spec:
  ...
  initContainers:
   - name: download-tools
     image: alpine:3
     command: [sh, -c]
     args:
       - wget -qO janus-go https://github.com/zepellin/janus-go/releases/download/v0.6.6/janus-v0.6.6-linux-amd64 && chmod +x janus-go && mv janus-go /janus-go/
     volumeMounts:
       - mountPath: /janus-go
         name: janus-go
  containers:
  - name: main-container
    ...
    volumeMounts:
    - mountPath: /usr/local/bin/janus-go
      name: janus-go
      subPath: janus-go

  volumes:
   - name: janus-go
     emptyDir: {}
```

Alternatively, use the published OCI image as a Kubernetes image volume (requires Kubernetes 1.31+ with the `ImageVolume` feature gate enabled):

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app-pod
spec:
  containers:
    - name: main-container
      image: my-app-image:latest
      volumeMounts:
        - mountPath: /usr/local/bin/janus-go
          name: janus-go
          subPath: janus-go
          readOnly: true
  volumes:
    - name: janus-go
      image:
        reference: ghcr.io/zepellin/janus-go-volume:v0.6.6
        pullPolicy: IfNotPresent
```

The OCI image contains two files at root — `/janus-go` and `/janus` (both are the same binary). Use `subPath` to pick whichever name you prefer:

```yaml
      volumeMounts:
        - mountPath: /usr/local/bin/janus-go
          name: janus-go
          subPath: janus-go   # or "janus"
          readOnly: true
```

### Image naming and tags

The OCI volume image is built by the [publish-volume-image](.github/workflows/publish-volume-image.yaml) workflow and published to GitHub Container Registry as:

```text
ghcr.io/<owner>/<repo>-volume
```

For this repository, that resolves to:

```text
ghcr.io/zepellin/janus-go-volume
```

The image is built for `linux/amd64` and `linux/arm64`.

Tag behavior:

- Release tags: `vX.Y.Z` (for example `v0.6.6`)
- Commit tags: `sha-<git-sha>`
- `latest`: published from the default branch

Examples:

```text
ghcr.io/zepellin/janus-go-volume:v0.6.6
ghcr.io/zepellin/janus-go-volume:sha-abc1234def56
ghcr.io/zepellin/janus-go-volume:latest
```

## Usage

Assuming pre-requisites for running the application have been met and AWS SDK configuration file in a following format exists:

```text
[profile my-aws-account]
credential_process = /usr/local/bin/janus-go -rolearn arn:aws:iam::123456789012:role/my-trusted-role
```

AWS clients such as AWS CLI or [AWS Terraform provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs) can now authenticate agains specified AWS profile and use AWS APIs.

```bash
aws --profile my-aws-account ec2 describe-instances
```

### Environment variables

| Variable | Description |
| --- | --- |
| `AWS_SESSION_IDENTIFIER` | STS role session name to use. Takes precedence over the value generated from GCP metadata (project ID + hostname), but not over the `-sessionid` flag. Must be 2-64 characters of `A-Za-z0-9+=,.@_-`. |
| `IDENTITY_TOKEN_AUDIENCE` | Audience of the requested Google identity token (defaults to `gcp`). Must match the audience configured in the AWS IAM role trust policy. Applies to GCE instance and service account tokens; the gcloud authorized user flow always uses the Google Cloud SDK audience. |

## Contributing

To contribute to Janus-go, follow these steps:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature_branch` ).
3. Make your changes.
4. Commit your changes (`git commit -am 'Add some feature'` ).
5. Push to the branch (`git push origin feature_branch` ).
6. Create a new Pull Request.

## License

This project uses the following license: [MIT](LICENSE).
