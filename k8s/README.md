# Deploy Mainflux on Kubernetes - WIP
Scripts to deploy Mainflux on Kubernetes (https://kubernetes.io). Work in progress. Not ready for deployment.

## Steps

### 1. Setup NATS

- Update `nats.conf` according to your needs.

- Create Kubernetes configmap to store NATS configuration:

```
kubectl create configmap nats-config --from-file=k8s/nats/nats.conf
```

- Deploy NATS:

```
kubectl create -f k8s/nats/nats.yml
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

### 3. Setup Clients service

- Deploy PostgreSQL service for Clients service to use:

```
kubectl create -f k8s/mainflux/clients-postgres.yml
```

- Deploy Clients service:

```
kubectl create -f k8s/mainflux/clients.yml
```

### 4. Setup Normalizer service

- Deploy Normalizer service:

```
kubectl create -f k8s/mainflux/normalizer.yml
```

### 5. Setup adapter services

- Deploy adapter service:

```
kubectl create -f k8s/mainflux/<adapter_service_name>.yml
```

### 6. Setup Dashflux

- Deploy Dashflux service:

```
kubectl create -f k8s/mainflux/dashflux.yml
```

### 7. Setup NginX Reverse Proxy for Mainflux Services

- Create TLS server side certificate and keys:

```
kubectl create secret tls mainflux-secret --key k8s/nginx/certs/mainflux-server.key --cert k8s/nginx/certs/mainflux-server.crt
```

- Create Kubernetes configmap to store NginX configuration:

```
kubectl create configmap mainflux-nginx-config --from-file=k8s/nginx/default.conf
```

- Deploy NginX service:

```
kubectl create -f k8s/nginx/nginx.yml
```

### 8. Configure Internet Access

Configure NAT on your Firewall to forward ports 80 (HTTP) and 443 (HTTPS) to mainflux-nginx service
