package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	Namespace                 string
	GitHubAppSecretName       string
	CaptainDomainConfigMap    string
	WatchDelay                time.Duration
	QRMintToken               string
	QRTTLSeconds              int64
	HTTPTimeout               time.Duration
	GitHubAPIVersionHeaderVal string
}

func main() {
	_ = godotenv.Load()
	cfg := loadConfig()

	ctx := context.Background()

	kubeClient, dynClient, err := newKubeClients()
	if err != nil {
		log.Fatalf("kubernetes client init failed: %v", err)
	}

	captainDomain, err := getCaptainDomain(ctx, kubeClient, cfg.Namespace, cfg.CaptainDomainConfigMap)
	if err != nil {
		log.Fatalf("failed to load CAPTAIN_DOMAIN: %v", err)
	}
	qrBase := "https://qr-code-generator." + captainDomain

	log.Printf("captain domain: %s", captainDomain)
	log.Printf("qr service base: %s", qrBase)

	processed := make(map[string]struct{}) // head_sha -> processed
	appsGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}

	ticker := time.NewTicker(cfg.WatchDelay)
	defer ticker.Stop()

	for {
		if err := processOnce(ctx, cfg, kubeClient, dynClient, appsGVR, qrBase, processed); err != nil {
			log.Printf("loop error: %v", err)
		}
		<-ticker.C
	}
}

func loadConfig() Config {
	ns := getenv("NAMESPACE", "glueops-core")
	secretName := getenv("GITHUB_APP_SECRET_NAME", "tenant-repo-creds")
	cmName := getenv("CAPTAIN_DOMAIN_K8S_CONFIGMAP_NAME", "glueops-captain-domain")

	delaySec := mustAtoi(getenv("WATCH_FOR_APPS_DELAY_SECONDS", "10"))
	mintToken := os.Getenv("QR_MINT_TOKEN")
	if mintToken == "" {
		log.Fatal("missing env var QR_MINT_TOKEN")
	}

	ttl := int64(mustAtoi(getenv("QR_TTL_SECONDS", "600"))) // default 10 minutes

	return Config{
		Namespace:                 ns,
		GitHubAppSecretName:       secretName,
		CaptainDomainConfigMap:    cmName,
		WatchDelay:                time.Duration(delaySec) * time.Second,
		QRMintToken:               mintToken,
		QRTTLSeconds:              ttl,
		HTTPTimeout:               15 * time.Second,
		GitHubAPIVersionHeaderVal: "2022-11-28",
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("invalid int %q: %v", s, err)
	}
	return i
}

func newKubeClients() (*kubernetes.Clientset, dynamic.Interface, error) {
	var rc *rest.Config
	var err error

	// Prefer in-cluster; fall back to kubeconfig
	rc, err = rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = home + "/.kube/config"
		}
		rc, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, nil, err
		}
	}

	kubeClient, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, nil, err
	}
	dynClient, err := dynamic.NewForConfig(rc)
	if err != nil {
		return nil, nil, err
	}
	return kubeClient, dynClient, nil
}

func getCaptainDomain(ctx context.Context, kube *kubernetes.Clientset, namespace, configMapName string) (string, error) {
	cm, err := kube.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if cm.Data == nil {
		return "", errors.New("configmap has no data")
	}
	d, ok := cm.Data["captain_domain"]
	if !ok || strings.TrimSpace(d) == "" {
		return "", errors.New("configmap missing key captain_domain")
	}
	return strings.TrimSpace(d), nil
}

func processOnce(
	ctx context.Context,
	cfg Config,
	kube *kubernetes.Clientset,
	dyn dynamic.Interface,
	appsGVR schema.GroupVersionResource,
	qrBase string,
	processed map[string]struct{},
) error {
	ul, err := dyn.Resource(appsGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list applications: %w", err)
	}

	items := ul.Items
	for i := range items {
		app := &items[i]
		if !hasOwnerReferences(app) {
			continue
		}

		ann := getAnnotations(app)
		if ann["preview_environment"] != "true" {
			continue
		}

		headSHA := ann["head_sha"]
		if headSHA == "" {
			continue
		}
		if _, ok := processed[headSHA]; ok {
			log.Printf("skipping already processed head_sha=%s app=%s", headSHA, app.GetName())
			continue
		}

		// Wait until the app is actually synced to that SHA
		if !syncRevisionsContainHeadSHA(app, headSHA) {
			log.Printf("waiting for sync to start: app=%s head_sha=%s", app.GetName(), headSHA)
			continue
		}

		health := getHealthStatus(app)
		if health != "Healthy" && health != "Degraded" {
			log.Printf("waiting for health: app=%s health=%s", app.GetName(), health)
			continue
		}

		externalURLs := getExternalURLs(app)
		if len(externalURLs) == 0 {
			log.Printf("no externalURLs yet: app=%s", app.GetName())
			continue
		}

		// Required GitHub metadata from annotations (same as your python)
		repo := ann["repository_name"]
		owner := ann["repository_organization"]
		prNum := ann["pull_request_number"]
		if repo == "" || owner == "" || prNum == "" {
			log.Printf("missing github metadata annotations for app=%s (owner=%q repo=%q pr=%q)", app.GetName(), owner, repo, prNum)
			continue
		}

		namespace := getDestinationNamespace(app)
		appName := app.GetName()

		// Build links
		captainDomain := strings.TrimPrefix(qrBase, "https://qr-code-generator.")
		// captainDomain is derived already; easier to compute from qrBase if you want:
		_ = captainDomain

		appLogsURL := getGrafanaURLLoki(qrBaseToCaptainDomain(qrBase), appName)
		appMetricsURL := getGrafanaURLMetrics(qrBaseToCaptainDomain(qrBase), namespace, appName)
		appArgoURL := getArgoCDApplicationURL(qrBaseToCaptainDomain(qrBase), appName)

		// Mint signed QR URLs for each external URL
		deploymentPreviewHTML := buildDeploymentPreviewHTML(ctx, cfg, qrBase, externalURLs)

		body := buildPRComment(ann, appArgoURL, deploymentPreviewHTML, appMetricsURL, appLogsURL)

		// Get GitHub installation token from k8s secret
		ghToken, err := getGitHubInstallationToken(ctx, cfg, kube)
		if err != nil {
			return fmt.Errorf("get github installation token: %w", err)
		}

		// Post PR comment
		if err := createGitHubPRComment(ctx, cfg, owner, repo, prNum, ghToken, body); err != nil {
			return fmt.Errorf("post pr comment: %w", err)
		}

		processed[headSHA] = struct{}{}
		log.Printf("SUCCESS: processed PR comment for app=%s head_sha=%s", appName, headSHA)
	}

	return nil
}

/* -------------------- App parsing helpers -------------------- */

func hasOwnerReferences(app *unstructured.Unstructured) bool {
	ors := app.GetOwnerReferences()
	return len(ors) > 0
}

func getAnnotations(app *unstructured.Unstructured) map[string]string {
	a := app.GetAnnotations()
	if a == nil {
		return map[string]string{}
	}
	return a
}

func syncRevisionsContainHeadSHA(app *unstructured.Unstructured, headSHA string) bool {
	revs, found, _ := unstructured.NestedStringSlice(app.Object, "status", "sync", "revisions")
	if !found || len(revs) == 0 {
		return false
	}
	for _, r := range revs {
		if r == headSHA {
			return true
		}
	}
	return false
}

func getHealthStatus(app *unstructured.Unstructured) string {
	s, _, _ := unstructured.NestedString(app.Object, "status", "health", "status")
	return s
}

func getExternalURLs(app *unstructured.Unstructured) []string {
	urls, found, _ := unstructured.NestedStringSlice(app.Object, "status", "summary", "externalURLs")
	if !found {
		return nil
	}
	// Filter empties
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		if strings.TrimSpace(u) != "" {
			out = append(out, u)
		}
	}
	return out
}

func getDestinationNamespace(app *unstructured.Unstructured) string {
	ns, _, _ := unstructured.NestedString(app.Object, "spec", "destination", "namespace")
	return ns
}

/* -------------------- Link builders (same as python) -------------------- */

func qrBaseToCaptainDomain(qrBase string) string {
	// qrBase is "https://qr-code-generator.<domain>"
	return strings.TrimPrefix(qrBase, "https://qr-code-generator.")
}

func getGrafanaURLPrefix(captainDomain string) string {
	return "https://grafana." + captainDomain
}

func getGrafanaURLLoki(captainDomain, appName string) string {
	return getGrafanaURLPrefix(captainDomain) + "/d/tBmi6B0Vz/loki-logs?orgId=1&var-workload=" +
		url.QueryEscape(appName) + "&from=now-3h&to=now"
}

func getGrafanaURLMetrics(captainDomain, namespace, appName string) string {
	return getGrafanaURLPrefix(captainDomain) +
		"/d/a164a7f0339f99e89cea5cb47e9be617/kubernetes-compute-resources-workload?var-datasource=Prometheus&var-cluster=&var-namespace=" +
		url.QueryEscape(namespace) + "&var-workload=" + url.QueryEscape(appName) +
		"&var-type=deployment&orgId=1"
}

func getArgoCDApplicationURL(captainDomain, appName string) string {
	return "https://argocd." + captainDomain + "/applications/" + url.PathEscape(appName)
}

/* -------------------- QR minting + comment building -------------------- */

func buildDeploymentPreviewHTML(ctx context.Context, cfg Config, qrBase string, externalURLs []string) string {
	if len(externalURLs) == 0 {
		return "Not available. No Ingress was configured."
	}

	var b strings.Builder
	for _, ext := range externalURLs {
		signedQR, err := mintSignedQRURL(ctx, cfg, qrBase, ext)
		if err != nil {
			// Don’t fail the whole PR comment; still show the URL
			log.Printf("mint qr failed for %q: %v", ext, err)
			_, _ = fmt.Fprintf(&b, `<details><summary>%s</summary><br><em>QR unavailable</em></details>`, htmlEscape(ext))
			continue
		}
		_, _ = fmt.Fprintf(
			&b,
			`<details><summary>%s</summary><br><img src="%s" width="100" height="100"></details>`,
			htmlEscape(ext),
			htmlEscape(signedQR),
		)
	}
	return b.String()
}

func mintSignedQRURL(ctx context.Context, cfg Config, qrBase string, target string) (string, error) {
	// Calls protected endpoint:
	// GET {qrBase}/v1/sign?u=<target>&ttl=<seconds>
	u, _ := url.Parse(qrBase + "/v1/sign")
	q := u.Query()
	q.Set("u", target)
	q.Set("ttl", strconv.FormatInt(cfg.QRTTLSeconds, 10))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.QRMintToken)

	client := &http.Client{Timeout: cfg.HTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("mint endpoint status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	signedPath := strings.TrimSpace(string(bodyBytes))
	if !strings.HasPrefix(signedPath, "/v1/qr?") {
		return "", fmt.Errorf("unexpected mint response: %q", signedPath)
	}

	// Full public QR image URL:
	return qrBase + signedPath, nil
}

func buildPRComment(ann map[string]string, appArgoURL, deploymentPreviewHTML, metricsURL, logsURL string) string {
	var b strings.Builder
	b.WriteString("|  Name | Link |\n|---------------------------------|------------------------|")

	headSHA := ann["head_sha"]
	b.WriteString(firstColumn("🔨", "Latest commit"))
	b.WriteString(htmlEscape(headSHA))
	b.WriteString(" |")

	b.WriteString(firstColumn("🦄", "Deployment Details"))
	b.WriteString("[ArgoCD](" + appArgoURL + ") |")

	b.WriteString(firstColumn("🖥️", "Deployment Preview"))
	b.WriteString(deploymentPreviewHTML)
	b.WriteString("|")

	b.WriteString(firstColumn("📊", "Metrics"))
	b.WriteString("[Grafana](" + metricsURL + ") |")

	b.WriteString(firstColumn("📜", "Logs"))
	b.WriteString("[Loki](" + logsURL + ") |")

	return b.String()
}

func firstColumn(emoji, text string) string {
	return "\n|<span aria-hidden=\"true\">" + emoji + "</span>  " + text + " |  "
}

// minimal escaping for summaries; GitHub handles some HTML, but keep safe-ish.
func htmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}

/* -------------------- GitHub auth + comment posting -------------------- */

type githubAppSecret struct {
	AppID            string
	InstallationID   string
	PrivateKeyPEM    string
	PrivateKeyParsed *rsa.PrivateKey
}

func getGitHubInstallationToken(ctx context.Context, cfg Config, kube *kubernetes.Clientset) (string, error) {
	sec, err := kube.CoreV1().Secrets(cfg.Namespace).Get(ctx, cfg.GitHubAppSecretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	appSecret, err := parseGitHubAppSecret(sec)
	if err != nil {
		return "", err
	}

	appJWT, err := createGitHubAppJWT(appSecret.PrivateKeyParsed, appSecret.AppID)
	if err != nil {
		return "", err
	}

	token, err := exchangeInstallationToken(ctx, cfg, appJWT, appSecret.InstallationID)
	if err != nil {
		return "", err
	}
	return token, nil
}

func parseGitHubAppSecret(sec *corev1.Secret) (githubAppSecret, error) {
	get := func(k string) (string, error) {
		v, ok := sec.Data[k]
		if !ok {
			return "", fmt.Errorf("secret missing key %q", k)
		}
		s := strings.TrimSpace(string(v))
		if s == "" {
			return "", fmt.Errorf("secret key %q is empty", k)
		}
		return s, nil
	}

	appID, err := get("githubAppID")
	if err != nil {
		return githubAppSecret{}, err
	}
	installID, err := get("githubAppInstallationID")
	if err != nil {
		return githubAppSecret{}, err
	}
	pemStr, err := get("githubAppPrivateKey")
	if err != nil {
		return githubAppSecret{}, err
	}

	priv, err := parseRSAPrivateKeyFromPEM(pemStr)
	if err != nil {
		return githubAppSecret{}, err
	}

	return githubAppSecret{
		AppID:            appID,
		InstallationID:   installID,
		PrivateKeyPEM:    pemStr,
		PrivateKeyParsed: priv,
	}, nil
}

func parseRSAPrivateKeyFromPEM(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM")
	}

	// Try PKCS#1
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	// Try PKCS#8
	keyI, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	key, ok := keyI.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

func createGitHubAppJWT(priv *rsa.PrivateKey, appID string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"iat": now.Add(-30 * time.Second).Unix(), // small clock skew buffer
		"exp": now.Add(10 * time.Minute).Unix(),  // GitHub max is 10 minutes
		"iss": appID,
		"jti": randomJTI(),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return t.SignedString(priv)
}

func randomJTI() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	// base64-url without padding manually:
	const enc = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	out := make([]byte, 22)
	// crude but fine; just make something non-empty and varied:
	for i := range out {
		out[i] = enc[int(b[i%16])%len(enc)]
	}
	return string(out)
}

func exchangeInstallationToken(ctx context.Context, cfg Config, appJWT, installationID string) (string, error) {
	u := "https://api.github.com/app/installations/" + installationID + "/access_tokens"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(nil))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("X-GitHub-Api-Version", cfg.GitHubAPIVersionHeaderVal)

	client := &http.Client{Timeout: cfg.HTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("installation token exchange failed status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var parsed struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return "", err
	}
	if parsed.Token == "" {
		return "", errors.New("github response missing token")
	}
	return parsed.Token, nil
}

func createGitHubPRComment(ctx context.Context, cfg Config, owner, repo, prNumber, token, body string) error {
	u := "https://api.github.com/repos/" + owner + "/" + repo + "/issues/" + prNumber + "/comments"

	payload := map[string]string{"body": body}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	// "token <...>" works with GitHub; "Bearer" also works for many endpoints.
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", cfg.GitHubAPIVersionHeaderVal)

	client := &http.Client{Timeout: cfg.HTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github comment failed status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

/* -------------------- tiny constant-time helper -------------------- */

// not currently used above, but handy if you later protect other endpoints:
func constTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		// subtle.ConstantTimeCompare requires same length; treat as not equal
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// also handy if you ever compare bearer headers:
func constTimeHeaderEqual(got, want string) bool {
	return hmac.Equal([]byte(got), []byte(want))
}
