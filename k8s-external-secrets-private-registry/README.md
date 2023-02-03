+++
summary: We show how to streamline the management of pull secrets of private repositories such as GitLab with a combination of External Secrets Operator and Doppler.
is_published: True
+++

# How to use Kubernetes External Secrets Operator with Private Container Registry

Most images we deploy to a Kubernetes cluster come from a private container registry.
Typically this registry is provided by the cloud platform that manages the Kubernetes cluster, and they are well integrated, so
you do not need to make extra efforts to make it work. But sometimes, the cloud platform does not have an integrated repository, or the company policy does not allow to use of the one provided by the platform. In that case, you have to provide
`imagePullSecret` in your Kubernetes manifest. [The Kubernetes documentation has a good description of how to do this](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/). However, that method can be further streamlined
with the [External Secrets Operator](https://external-secrets.io/v0.7.2/) and its templating capability, so you do not need to manage the secret, the dockerconfig json file and its base64 encoding manually. In the following, we show how
to combine the use of [Doppler](https://www.doppler.com/) security operation platform and the External Secret Operator to deploy images to Kubernetes from the Gitlab private container registry.

In the following, we assume that:

-   you set up your Doppler projects, environments (they call it configs), and its command line client `doppler`,
-   you set up the Gitlab container registry and pushed an image there,
-   you have a Kubernetes cluster configured to be accessible using `kubectl` and `helm`,
-   you have a namespace in Kubernetes called `test` where you want to deploy,
-   and finally, your shell is configured to not save commands to the history that starts with space

## Setting up External Secrets Operator with Doppler

First - if you have not already - you should install the External Secrets Operator using `helm`:

```shell
helm repo add external-secrets https://charts.external-secrets.io

helm install external-secrets \
   external-secrets/external-secrets \
    -n external-secrets \
    --create-namespace \
    --set installCRDs=true
```

Then you create a `SecretStore` or `ClusterSecretStore` custom resource configured to use Doppler.
We prefer `SecretStore` since it is a namespaced resource, allowing better separation and access control.
The `SecretStore` needs a Kubernetes `Secret` that contains a Doppler access token. Every Doppler access token is
scoped to a specific project and config (environment) combination. You can create one named by, for example, `external-secret-operator` using doppler's CLI:

```shell
❯ doppler --project <project name> --config <config name> configs tokens create external-secret-operator --plain
dp.st.XXXX
```

Using the returned token, you can create a Kubernetes secret that holds this token:

```shell
❯  kubectl create secret generic doppler-auth-token --namespace test --from-literal dopplerToken="dp.st.XXXX"
```

(be aware of the extra space at the beginning of the command to not save it to history)

Then to create a `SecretStore`:

```yaml, filename=secret-store.yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
    name: doppler-secret-store
    namespace: test
spec:
    provider:
        doppler:
            auth:
                secretRef:
                    dopplerToken:
                        name: doppler-auth-token
                        key: dopplerToken
```

## Sync Gitlab Docker Registry Credentials from Doppler to Kubernetes

First, you create a deploy token on Gitlab. You can do that under the Project page / Settings / Repository / Deploy Tokens. The scope should be `read_repository`. Put the created access token username and password to Doppler under `GITLAB_DEPLOY_TOKEN_USERNAME` and `GITLAB_DEPLOY_TOKEN_PASSWORD`, respectively. Further, add the following secrets to Doppler:

-   `GITLAB_REGISTRY_HOST`: the hostname of the Gitlab Private Repository. Typically `registry.gitlab.com`.
-   `GITLAB_REGISTRY_EMAIL`: use your GitLab email address

The following manifest creates an `ExternalSecret` named `registry-credetial` that uses the previously defined Doppler secrets and the `SecretStore` to create and sync a Kubernetes Secret named `gitlab-registry-credential` with properly formatted `.dockerconfigjson` data using Go template. You can adjust the `refreshInterval` during development to lower value to get synced faster.

```yaml, filename=gitlab-pull-secret.yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
    name: registry-credential
    namespace: test
spec:
    refreshInterval: 3600s
    secretStoreRef:
        kind: SecretStore
        name: doppler-secret-store
    target:
        name: gitlab-registry-credential
        template:
            type: kubernetes.io/dockerconfigjson
            data:
                .dockerconfigjson: |
                    {
                      "auths": {
                        "{{ .registry }}": {
                          "username": "{{ .username }}",
                          "password": "{{ .password }}",
                          "email": "{{ .registryEmail }}",
                          "auth": "{{ list .username .password | join ":" | b64enc }}"
                        }
                      }
                    }
    data:
        - secretKey: registry
          remoteRef:
              key: GITLAB_REGISTRY_HOST
        - secretKey: registryEmail
          remoteRef:
              key: GITLAB_REGISTRY_EMAIL
        - secretKey: username
          remoteRef:
              key: GITLAB_DEPLOY_TOKEN_USERNAME
        - secretKey: password
          remoteRef:
              key: GITLAB_DEPLOY_TOKEN_PASSWORD
```

# Using Secret to deploy from private repository

The following manifest show how to create a `Deployment` that uses the previously defined `gitlab-registry-credential` Kubernetes secret as a `imagePullSecrets`. Important to note that the `imagePullSecret` name is the target secret name of the `ExternalSecret`, not the external secret itself.

```yaml, filename=deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
    name: pionlab-server
    namespace: test
spec:
    replicas: 1
    selector:
        matchLabels:
            app: pionlab-server
    template:
        metadata:
            labels:
                app: pionlab-server
        spec:
            containers:
                - name: pionlab-server
                  image: registry.gitlab.com/pionlab/pionlab:latest
                  imagePullPolicy: Always
                  args:
                      - ./scripts/runserver.sh
                  envFrom:
                      - secretRef:
                            name: django-environment-vars
            imagePullSecrets:
                - name: gitlab-registry-credential
```

You can find other ways to utilize Doppler configs as environment variables for deployment in [Doppler documentation](https://docs.doppler.com/docs/external-secrets-provider).
