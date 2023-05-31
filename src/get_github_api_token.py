import time
from base64 import b64decode

import jwt
import requests
from kubernetes import client


def get_github_app_kubernetes_secret(
    k8s_v1_api: client.api.core_v1_api.CoreV1Api,
    secret_name: str,
    secret_namespace: str,
) -> dict:
    """Retrieves and decodes components of a GitHub app secret
        from Kubernetes Repo Creds secret

    Args:
        k8s_v1_api (client.api.core_v1_api.CoreV1Api): Kubernetes v1 API Client
        secret_name (str): Name of the repo creds secret.
        secret_namespace (str): Namespace where repo creds secret is located

    Returns:
        dict: Dictionary of components of GitHub app secret
    """
    secret = k8s_v1_api.read_namespaced_secret(secret_name, secret_namespace)
    
    def decode(secret_data: str) -> str:
        return b64decode(secret_data).decode('utf-8')
    
    return {
        'app_id': decode(secret.data['githubAppID']),
        'app_installation_id': decode(secret.data['githubAppInstallationID']),
        'app_private_key': decode(secret.data['githubAppPrivateKey']),
    }

def get_jwt(
    pem: str,
    app_id: str,
) -> str:
    """Create jwt from private key.

    Args:
        pem (str): string of private key
        app_id (str): ID of GitHub application requesting key

    Returns:
        str: JWT to request GitHub app API token
    """
    payload = {
        # Issued at time
        'iat': int(time.time()),
        # JWT expiration time (10 minutes maximum)
        'exp': int(time.time()) + 600,
        # GitHub App's identifier
        'iss': app_id
    }
    
    signing_key = jwt.jwk_from_pem(pem.encode())
    
    # Create JWT
    jwt_instance = jwt.JWT()
    encoded_jwt = jwt_instance.encode(payload, signing_key, alg='RS256')

    return encoded_jwt
    
def get_github_api_token(
    k8s_v1_api: client.api.core_v1_api.CoreV1Api,
    secret_name: str,
    secret_namespace: str,
) -> str:
    """Create GitHub API token from Kubernetes Secret

    Args:
        k8s_v1_api (client.api.core_v1_api.CoreV1Api): Kubernetes v1 API Client
        secret_name (str): Name of the repo creds secret.
        secret_namespace (str): Namespace where repo creds secret is located

    Returns:
        str: GitHub API token
    """
    app_secret = get_github_app_kubernetes_secret(
        k8s_v1_api=k8s_v1_api,
        secret_name=secret_name,
        secret_namespace=secret_namespace,
    )
    encoded_jwt = get_jwt(
        pem=app_secret['app_private_key'],
        app_id=app_secret['app_id'],
    )
    headers = {
        "Accept": "application/vnd.github+json",
        "Authorization": f"Bearer {encoded_jwt}",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    url = f"https://api.github.com/app/installations/{app_secret['app_installation_id']}/access_tokens"

    response = requests.post(
        url,
        headers=headers,
    )
    data = response.json()
    
    return data['token']
