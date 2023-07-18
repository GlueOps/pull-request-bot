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

## Running the app

- Ensure you have the following set in your ```.env``` file (at **root** foolder):

```bash
export GITHUB_TOKEN=<some-value>
```

For cloud specific setup (to be authenticated to the captain cluster), check [here](https://github.com/GlueOps/terraform-module-cloud-aws-kubernetes-cluster/wiki)

- Then run

```python
python main.py
```