# Janus-go

## Description
Janus-go is a AWS CLI external source authentication program for use with Google Cloud GKE workload identity or GCE VM identity. It is designed to allow authenticating AWS IAM role from Google Cloud environments (such as GKE cluster or GCE VM instance) without the need of generating long term AWS credentials.

This project was inspired by [Janus](https://github.com/doitintl/janus), a python implementation of the same authentication flow. This project was written in go for easier installation and usage of the program where a single binary is implementation is better suited (such as inside of existing container running on kubernetes).

## Prerequisites
1. The environment in which the program is running has to be able to provide Google Cloud Identity token from Google Cloud metadata server. This can be achieved either by running on GCE VM instance or as a GKE workload with workload identity enabled
2. An AWS IAM role is created with a [trust policy specifying the Google Cloud IAM identity](https://aws.amazon.com/blogs/security/access-aws-using-a-google-cloud-platform-native-workload-identity/) used by VM instance or GKE workload identity from step 1. 
## Installation
### Locally
Download appropriate release for your OS and achitecture from the project's release page.

```bash
wget -qO janus-go https://github.com/zepellin/janus-go/releases/download/v0.2.1/janus-v0.2.1-linux-amd64 && chmod +x janus-go
```
### Inside Kubernetes pod
To use the binary inside of Kubernetes pod, download the binary using init container and mount the binary path inside of your main container:

```
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
       - wget -qO janus-go https://github.com/zepellin/janus-go/releases/download/v0.2.1/janus-v0.2.1-linux-amd64 && chmod +x janus-go && mv janus-go /janus-go/
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
## Usage
Assuming pre-requisites for running the application have been met and AWS SDK configuration file in a following format exists:

```
[profile my-aws-account]
credential_process = /usr/local/bin/janus-go -rolearn arn:aws:iam::123456789012:role/my-trusted-role
```
AWS clients such as AWS CLI or [AWS Terraform provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs) can now authenticate agains specified AWS profile and use AWS APIs.

```
aws --profile my-aws-account ec2 describe-instances
```
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

