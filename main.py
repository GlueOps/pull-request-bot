import logging
import os
import time

import requests
from kubernetes import client, config

from src.get_github_api_token import get_github_api_token
from src.json_log_formatter import JsonFormatter

#=== configure logging
# json formatter
json_formatter = JsonFormatter()

# stream handler
stream_handler = logging.StreamHandler()
stream_handler.setFormatter(json_formatter)
stream_handler.setLevel(os.getenv(
    'PYTHON_LOG_LEVEL',ss
    'INFO'
))

# configure logger
logger = logging.getLogger('PULL_REQUEST_BOT')
logger.setLevel(os.getenv(
    'PYTHON_LOG_LEVEL',
    'INFO'
))
logger.addHandler(stream_handler)

# setting cluster config
try:
    config.load_incluster_config()
except Exception as e:
    logger.warning(f'Error loading in-cluster k8s config: {e}')
    try:
        logger.info('Using local Kubeconfig (not in-cluster)')
        config.load_kube_config()
    except Exception:
        logger.exception('Failed to load Kubeconfig from cluster, local file')

# configure kubernetes api clients
v1 = client.CoreV1Api()
custom_api = client.CustomObjectsApi()

# set app constants
NAMESPACE = os.getenv(
    'NAMESPACE',
    'glueops-core'
)
GITHUB_APP_SECRET_NAME = os.getenv(
    'GITHUB_APP_SECRET_NAME',
    'tenant-repo-creds'
)
CAPTAIN_DOMAIN_K8S_CONFIGMAP_NAME = os.getenv(
    'CAPTAIN_DOMAIN_K8S_CONFIGMAP_NAME',
    'glueops-captain-domain'
)
WATCH_FOR_APPS_DELAY_SECONDS =int(os.getenv(
    'WATCH_FOR_APPS_DELAY_SECONDS',
    '10'
))


def get_captain_domain():
    configmap = v1.read_namespaced_config_map(
        CAPTAIN_DOMAIN_K8S_CONFIGMAP_NAME, NAMESPACE)
    return configmap.data['captain_domain']

try:
    CAPTAIN_DOMAIN = get_captain_domain()
except:
    logger.exception('Failed to load CAPTAIN_DOMAIN')

def main():
    commits_processed = []

    # Continuously watch for new ArgoCD applications
    while True:
        # Get the updated list of ArgoCD applications
        apps = custom_api.list_cluster_custom_object(
            'argoproj.io',
            'v1alpha1',
            'applications'
        )
        # Filter for ArgoCD applications created by an ApplicationSet
        new_apps = [
            app for app in apps['items']
            if app.get('metadata', {}).get('ownerReferences', [])
        ]

        git_provider = ""
        git_commit_metadata = ""
        external_urls = ""
        app_name = ""
        namespace = ""

        # Check each new application
        for app in new_apps:
            if app['metadata']['annotations'].get('preview_environment') == 'true':
                logger.info(f'OK. This app has the annotation preview_environment == true : {app["metadata"]["name"]}')
                if app['metadata']['annotations']['head_sha'] not in commits_processed:
                    # Check if the application was created by an ApplicationSet
                    owner_refs = app['metadata']['ownerReferences']
                    appset_created = any(
                        ref['kind'] == 'ApplicationSet' for ref in owner_refs
                    )
                    app_name = app['metadata']['name']
                    namespace = app['spec']['destination']['namespace']
                    # if the app was created by an ApplicationSet, get the ApplicationSet name and git commit metadata
                    if appset_created:
                        appset_name = app['metadata']['ownerReferences'][0]['name']
                        git_commit_metadata = app['metadata']['annotations']

                        git_provider = {
                            "github": {
                                "repo": git_commit_metadata["repository_name"],
                                "owner": git_commit_metadata["repository_organization"]
                            }
                        }

                    # Check if the application has an external URL defined in its status
                    if app.get('status', {}).get('summary', {}).get('externalURLs', []):
                        external_urls = app['status']['summary']['externalURLs']
                        has_external_url = any(url for url in external_urls)
                    else:
                        has_external_url = False

                    if has_external_url:
                        app_logs_url = get_grafana_url_loki(app_name)
                        app_metrics_url = get_grafana_url_metrics(
                            namespace,
                            app_name
                        )
                        app_argocd_url = get_argocd_application_url(app_name)
                        
                        pr_comment = get_comment(
                            git_commit_metadata,
                            app_name,
                            app_argocd_url,
                            external_urls,
                            app_logs_url,
                            app_metrics_url
                        )
                        git_provider_api_token = get_github_api_token(
                            k8s_v1_api=v1,
                            secret_name=GITHUB_APP_SECRET_NAME,
                            secret_namespace=NAMESPACE
                        )         
                        try:
                            r = update_pr(
                                git_provider,
                                git_commit_metadata,
                                pr_comment,
                                git_provider_api_token
                            )

                            r.raise_for_status()

                            commits_processed.append(
                                app['metadata']['annotations']['head_sha']
                            )
                            logger.debug(f'updated pr comment: {r.json()}')
                            logger.info(f'SUCCESS. Just processed PR comment for: {app["metadata"]["name"]}')
                        except:
                            logger.exception(f'Failed to process pr comment: {r.json()}')

                else:
                    logger.info(
                        f'Skipping. Already processed: {app["metadata"]["name"]} '
                        f'{app["metadata"]["annotations"]["head_sha"]}'
                    )
            else:
                logger.info(f'SKIPPING. This app does not have the annotation preview_environment == true : {app["metadata"]["name"]}')
        # Sleep for some time before checking again
        time.sleep(WATCH_FOR_APPS_DELAY_SECONDS)

def get_grafana_url_prefix():
    return "https://grafana." + CAPTAIN_DOMAIN


def get_grafana_url_loki(app_name):
    return get_grafana_url_prefix() + "/d/tBmi6B0Vz/loki-logs?orgId=1&var-workload=" + app_name + "&from=now-3h&to=now"


def get_grafana_url_metrics(namespace, app_name):
    return get_grafana_url_prefix()+"/d/a164a7f0339f99e89cea5cb47e9be617/kubernetes-compute-resources-workload?var-datasource=Prometheus&var-cluster=&var-namespace="+namespace+"&var-workload="+app_name+"&var-type=deployment&orgId=1"


def get_argocd_application_url(app_name):
    return "https://argocd." + CAPTAIN_DOMAIN + "/applications/" + app_name


def update_pr(git_provider, git_commit_metadata, pr_comment, git_provider_api_token):
    if 'github' in git_provider:
        github_pr_url = 'https://api.github.com/repos/' + \
            git_provider['github']['owner'] + '/' + git_provider['github']['repo'] + \
            '/issues/' + \
            git_commit_metadata['pull_request_number'] + '/comments'
        headers = {'Authorization': 'token ' + git_provider_api_token,
                   'Content-Type': 'application/json'}

        payload = {
            'body': pr_comment
        }

        response = requests.post(github_pr_url, headers=headers, json=payload)
        return response

def get_first_column(emoji, text):
    return '\n|<span aria-hidden=\"true\">' + emoji + '</span>  ' + text + ' |  '

def get_comment(git_commit_metadata, app_name, app_argocd_url, external_urls, app_logs_url, app_metrics_url):
    body = '|  Name | Link |\n|---------------------------------|------------------------|'
    body += get_first_column("🔨", "Latest commit") + git_commit_metadata['head_sha'] + ' |'
    body += get_first_column("🦄", "Deployment Details") + '[ArgoCD](' + app_argocd_url + ') |'
    body += get_first_column("🖥️", "Deployment Preview") + '[' + external_urls[0] + '](' + external_urls[0] + ') |'
    body += get_first_column("📊", "Metrics") + '[Grafana](' + app_metrics_url + ') |'
    body += get_first_column("📜", "Logs") + '[Loki](' + app_logs_url + ') |'
    qr_code_url = f'https://qr-code-generator.{get_captain_domain()}/v1/qr?url={external_urls[0]}'
    body += get_first_column("📱", "Preview on mobile") + f'<img src="{qr_code_url}" width="150" height="150">|'

    return body


if __name__ == '__main__':
    main()
