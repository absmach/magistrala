# Deploy Mainflux on Kubernetes - WIP
Scripts to deploy Mainflux on Kubernetes (https://kubernetes.io). Work in progress. Not ready for deployment.

## Steps

### 1. Setup PosgreSQL

- Create Persistent Volume for PosgreSQL to store data to.

```bash
kubectl create -f 1-mainflux-postgres-persistence.yml
```

- Claim Persistent Volume

```bash
kubectl create -f 2-mainflux-postgres-claim.yml
```

- Create PosgreSQL Pod

```bash
kubectl create -f 3-mainflux-postgres-pod.yml
```

- Create PosgreSQL Service

```bash
kubectl create -f 4-mainflux-postgres-service.yml
```

### 2. Setup NATS

- Change `nats.conf` according to your needs.

Create a Kubernetes configmap to store it:

```bash
kubectl create configmap nats-config --from-file nats.conf
```

- Deploy NATS:

```bash
kubectl create -f nats.yml
```

### 3. Setup Mainflux Services

- Create Manager Service

```bash
kubectl create -f 1-mainflux-manager.yml
```

- Create HTTP Service

```bash
kubectl create -f 2-mainflux-http.yml

```

- Create CoAP Service

```bash
kubectl create -f 4-mainflux-coap.yml
```

- Create Normalizer Service

```bash
kubectl create -f 5-mainflux-normalizer.yml
```

### 4. Setup Dashflux Services

- Create Dashflux Deployment and Service

```bash
kubectl create -f mainflux-dashflux.yaml
```

### 5. Setup NginX Reverse Proxy for Mainflux Services

- Create TLS server side certificate and keys

```bash
cd certs
kubectl create secret tls mainflux-secret --key mainflux-server.key --cert mainflux-server.crt
```

- Create Config Map based on the default.conf file.

```bash
cd ..
kubectl create configmap mainflux-nginx-config --from-file=default.conf
```

- Create Deployment and Service from mainflux-dashflux.yaml file.

```bash
kubectl create -f mainflux-nginx.yaml
```

### 6. Configure Internet Access

Configure NAT on your Firewall to forward ports 80 (HTTP) and 443 (HTTPS) to mainflux-nginx service
