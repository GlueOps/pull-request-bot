import requests
import time

from kubernetes import client, config
import os
import base64


try:
    config.load_incluster_config()
except Exception as e:
    print("Error loading in-cluster k8s config: {0}".format(e))
    config.load_kube_config()
    print("Using local Kubeconfig (not in-cluster)")

v1 = client.CoreV1Api()
custom_api = client.CustomObjectsApi()


NAMESPACE = os.environ.get('NAMESPACE') or "glueops-core"
API_TOKEN_K8S_SECRET_NAME = os.environ.get(
    'API_TOKEN_K8S_SECRET_NAME') or "git-provider-api-token"
CAPTAIN_DOMAIN_K8S_CONFIGMAP_NAME = os.environ.get(
    'CAPTAIN_DOMAIN_K8S_CONFIGMAP_NAME') or "glueops-captain-domain"


def get_api_token():
    secret = v1.read_namespaced_secret(API_TOKEN_K8S_SECRET_NAME, NAMESPACE)
    return base64.b64decode(secret.data["token"]).decode("utf-8")


GIT_PROVIDER_API_TOKEN = get_api_token()


def get_captain_domain():
    configmap = v1.read_namespaced_config_map(
        CAPTAIN_DOMAIN_K8S_CONFIGMAP_NAME, NAMESPACE)
    return configmap.data["captain_domain"]


CAPTAIN_DOMAIN = get_captain_domain()


def main():
    commits_processed = []

    # Continuously watch for new Argo CD applications
    while True:
        # Get the updated list of Argo CD applications
        apps = custom_api.list_cluster_custom_object(
            "argoproj.io", "v1alpha1", "applications")

        # Compare the initial and updated lists of Argo CD applications
        new_apps = [app for app in apps["items"] if app.get(
            "metadata", {}).get("ownerReferences", [])]

        git_provider = ""
        git_commit_metadata = ""
        external_urls = ""
        app_name = ""
        namespace = ""
#
        # Check each new application
        for app in new_apps:
            if app["metadata"]["annotations"]['head_sha'] not in commits_processed:
                # Check if the application was created by an ApplicationSet
                owner_refs = app["metadata"]["ownerReferences"]
                appset_created = any(
                    ref["kind"] == "ApplicationSet" for ref in owner_refs)
                app_name = app["metadata"]["name"]
                namespace = app["spec"]["destination"]["namespace"]
                # if the app was created by an ApplicationSet, get the ApplicationSet name and git commit metadata
                if appset_created:
                    appset_name = app["metadata"]["ownerReferences"][0]["name"]
                    git_commit_metadata = app["metadata"]["annotations"]
                    git_provider = git_provider_info(appset_name)

                # Check if the application has an external URL defined in its status
                if app.get("status", {}).get("summary", {}).get("externalURLs", []):
                    external_urls = app["status"]["summary"]["externalURLs"]
                    has_external_url = any(url for url in external_urls)
                else:
                    has_external_url = False

                if has_external_url:
                    app_logs_url = get_grafana_url_loki(app_name)
                    app_metrics_url = get_grafana_url_metrics(
                        namespace, app_name)
                    app_argocd_url = get_argocd_application_url(app_name)
                    pr_comment = get_comment(
                        git_commit_metadata, app_name, app_argocd_url, external_urls, app_logs_url, app_metrics_url)
                    update_pr(git_provider, git_commit_metadata, pr_comment)
                    commits_processed.append(
                        app["metadata"]["annotations"]['head_sha'])
            else:
                print("Skipping. Already processed: " + app["metadata"]["name"] +
                    " " + app["metadata"]["annotations"]['head_sha'])
            # Sleep for some time before checking again
        time.sleep(10)


def git_provider_info(appset_name):
    apps_sets = custom_api.list_cluster_custom_object(
        "argoproj.io", "v1alpha1", "applicationsets")
    for app_set in apps_sets['items']:
        if app_set['metadata']['name'] == appset_name:
            if "pullRequest" in app_set['spec']['generators'][0]:
                return app_set['spec']['generators'][0]['pullRequest']


def get_grafana_url_prefix():
    return "https://grafana." + CAPTAIN_DOMAIN


def get_grafana_url_loki(app_name):
    return get_grafana_url_prefix() + "/d/tBmi6B0Vz/loki-logs?orgId=1&var-workload=" + app_name + "&from=now-3h&to=now"


def get_grafana_url_metrics(namespace, app_name):
    return get_grafana_url_prefix()+"/d/a164a7f0339f99e89cea5cb47e9be617/kubernetes-compute-resources-workload?var-datasource=Prometheus&var-cluster=&var-namespace="+namespace+"&var-workload="+app_name+"&var-type=deployment&orgId=1"


def get_argocd_application_url(app_name):
    return "https://argocd." + CAPTAIN_DOMAIN + "/applications/" + app_name


def update_pr(git_provider, git_commit_metadata, pr_comment):
    if 'github' in git_provider:
        github_pr_url = 'https://api.github.com/repos/' + \
            git_provider['github']['owner'] + '/' + git_provider['github']['repo'] + \
            '/issues/' + \
            git_commit_metadata['pull_request_number'] + '/comments'
        headers = {"Authorization": "token " + GIT_PROVIDER_API_TOKEN,
                   'Content-Type': 'application/json'}

        payload = {
            'body': pr_comment
        }

        response = requests.post(github_pr_url, headers=headers, json=payload)

def get_first_column(emoji, text):
    return '\n|<span aria-hidden=\"true\">' + emoji + '</span>  ' + text + ' |  '


def get_comment(git_commit_metadata, app_name, app_argocd_url, external_urls, app_logs_url, app_metrics_url):
      body = '|  Name | Link |\n|---------------------------------|------------------------|'
      body += get_first_column("🔨", "Latest commit") + git_commit_metadata['head_sha'] + ' |'
      body += get_first_column("🦄", "Deployment Details") + '[ArgoCD](' + app_argocd_url + ') |'
      body += get_first_column("🖥️", "Deployment Preview") + '[' + external_urls[0] + '](' + external_urls[0] + ') |'
      body += get_first_column("📊", "Metrics") + '[Grafana](' + app_metrics_url + ') |'
      body += get_first_column("📜", "Logs") + '[Loki](' + app_logs_url + ') |'
      body += get_first_column("📱", "Preview on mobile") + '<img src="https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=' + external_urls[0] + '">|'
      return body

if __name__ == '__main__':
    main()
