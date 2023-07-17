# pr-bot

## Requirements

- You need a k8s cluster.
- You need to deploy the resources below before to the `glueops-core` cluster
- You need a git provider api token (Ex. Github Personal Access Token)
- You need a captain domain
  
```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: git-provider-api-token
type: Opaque
data:
  token: <base64 encoded github token>
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: glueops-captain-domain
data:
  captain_domain: <example: nonprod.tenant.glueopshosted.rocks>
---
```

## Running the main.py file

- Ensure you have the following set in your ```.env``` file (at root foolder):

```bash
export GITHUB_TOKEN=<some-value>
aws eks update-kubeconfig --region us-west-2 --name captain-cluster --role-arn arn:aws:iam::<some-value>:role/captain-role
```
