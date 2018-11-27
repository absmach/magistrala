# Deploy Mainflux on Kubernetes - WIP
Scripts to deploy Mainflux on Kubernetes (https://kubernetes.io). Work in progress. Not ready for deployment.

## Steps

### 1. Setup NATS

- To setup NATS cluster on k8s we recommend using [NATS operator](https://github.com/nats-io/nats-operator). NATS cluster should be deployed on namespace `nats-io` under the name `nats-cluster`.

### 2. Setup gRPC services Istio sidecar

- To load balance gRPC services we recommend using [Istio](https://istio.io/docs/setup/kubernetes/download-release/) sidecar. In order to use automatic inject you should run following command:

```
kubectl create -f k8s/mainflux/namespace.yml
```

### 2. Setup Users service

- Deploy PostgreSQL service for Users service to use:

```
kubectl create -f k8s/mainflux/users-postgres.yml
```

- Deploy Users service:

```
kubectl create -f k8s/mainflux/users.yml
```

### 3. Setup Things service

- Deploy PostgreSQL service for Things service to use:

```
kubectl create -f k8s/mainflux/things-postgres.yml
```

- Deploy Things service:

```
kubectl create -f k8s/mainflux/things.yml
```

### 4. Setup Normalizer service

- Deploy Normalizer service:

```
kubectl create -f k8s/mainflux/normalizer.yml
```

### 5. Setup adapter services

- Deploy adapter service:

```
kubectl create -f k8s/mainflux/tcp-services.yml
kubectl create -f k8s/mainflux/<adapter_service_name>.yml
```

### 6. Setup Dashflux

- Deploy Dashflux service:

```
kubectl create -f k8s/mainflux/dashflux.yml
```

### 7. Configure Internet Access

Configure NAT on your Firewall to forward ports 80 (HTTP) and 443 (HTTPS) to nginx ingress service
