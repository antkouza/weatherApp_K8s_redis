# 🛠️ Kubernetes & Redis Setup Guide (`SETUP.md`)

This guide outlines the step-by-step process for migrating full-stack application https://github.com/antkouza/weatherApp_Angular_and_Go from a traditional Docker Compose environment to a local Kubernetes (**Kind**) cluster utilizing a centralized **Redis distributed cache**.

## 🧭 Step 1: Cluster Infrastructure Setup

### 1. Install kubectl (The Kubernetes CLI)

Before installing Kind, you need `kubectl`. This is the command-line tool you will use to talk to your cluster (deploy apps, check logs, etc.).

Run these commands inside your WSL2 Ubuntu terminal:

```bash
# 1. Download the latest stable kubectl binary
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

# 2. Make it executable
chmod +x ./kubectl

# 3. Move it to your local bin directory so you can run it from anywhere
sudo mv ./kubectl /usr/local/bin/kubectl

# 4. Verify the installation
kubectl version --client

```

### 2. Install Kind

Now, let's install the actual Kind binary into your WSL2 Ubuntu environment.

Run these commands in your WSL2 Ubuntu terminal:

```bash
# 1. Download the Kind binary for AMD64 Linux
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.22.0/kind-linux-amd64

# 2. Make it executable
chmod +x ./kind

# 3. Move it to your local bin directory
sudo mv ./kind /usr/local/bin/kind

# 4. Verify the installation
kind version

```

### 3. Create Your First Cluster

Because we need to access the Angular frontend from a Windows browser on port `4200` and the API on port `8080`, we tell Kind to map those ports from the cluster containers to your host machine right now.

Create a configuration file named `kind-config.yaml`:

```bash
nano kind-config.yaml

```

Paste this configuration inside it (this maps NodePorts `30000` and `30080` from the cluster to your host machine ports `4200` and `8080` respectively):

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30000
    hostPort: 4200
    listenAddress: "0.0.0.0"
  - containerPort: 30080
    hostPort: 8080
    listenAddress: "0.0.0.0"

```

Spin up the cluster using this config:

```bash
kind create cluster --config kind-config.yaml --name weather-cluster

```

### 4. Verify Everything is Alive

Once Kind finishes pulling the node images and bootstrapping the cluster, run this command to make sure `kubectl` can see it:

```bash
kubectl cluster-info

```

> **Expected Output:**
> * Kubernetes control plane is running at `[https://127.0.0.1:35311](https://127.0.0.1:35311)`
> * CoreDNS is running at `[https://127.0.0.1:35311/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy](https://127.0.0.1:35311/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy)`
> 
> 

Now, verify the single controller node is healthy:

```bash
kubectl get nodes

```

```text
NAME                            STATUS   ROLES           AGE   VERSION
weather-cluster-control-plane   Ready    control-plane   43m   v1.29.2

```

You should see a single node named `weather-cluster-control-plane` with a status of **Ready**. The cluster is completely active, and ports `4200` and `8080` are now being routed directly into your local Kubernetes controller.

---

## 🧭 Step 2: Preparing our Images for Kubernetes

Normally, when you deploy an app to Kubernetes, K8s goes out to Docker Hub or GitHub Packages to download the images. Because we are working entirely locally on a development machine, we build our images locally and then load them directly into Kind's internal image registry.

### 1. Build the Frontend and Backend Images Locally

First, ensure your Go backend code contains the official Redis driver package by running this inside the `/backend` directory:

```bash
go get github.com/redis/go-redis/v9

```

Now, build standard production Docker images using your individual Dockerfiles from the root directory:

```bash
# Build the backend image tagged as 'weather-backend:v1'
docker build -t weather-backend:v1 ./backend

# Build the frontend image tagged as 'weather-frontend:v1'
docker build -t weather-frontend:v1 ./frontend

```

### 2. Push the Images into the Kind Cluster

Right now, Kubernetes does not know these images exist on our machine. We need to tell Kind to copy these images from our host Docker environment into the cluster container.

```bash
# Load the backend image into Kind
kind load docker-image weather-backend:v1 --name weather-cluster

# Load the frontend image into Kind
kind load docker-image weather-frontend:v1 --name weather-cluster

```

### 3. Verify Kind Received the Images

To guarantee that Kubernetes has local access to these images, run this command to inspect the cluster's internal package node:

```bash
docker exec -it weather-cluster-control-plane ctr -n k8s.io images list | grep weather

```

You should see both `weather-backend:v1` and `weather-frontend:v1` printed out safely in the terminal window.

---

## 🧭 Step 3: Writing Kubernetes Manifests (YAML)

In Kubernetes, we don't use a single file like `docker-compose.yml`. Instead, we write declarative configurations called **Manifests**.

Create a directory structure to hold these manifests cleanly in your project root folder:

```bash
mkdir k8s

```

We will implement our infrastructure across four distinct pieces inside the `/k8s` directory:

### 1. The Secret (API Key Security)

Kubernetes doesn't read `.env` files directly. We use a `Secret` object. Secrets expect values to be base64-encoded to prevent character corruption and accidental plaintext leaks.

1. Generate your base64-encoded API key by running this command in your terminal (replace `your_real_openweather_api_key` with your actual key):
```bash
echo -n "your_real_openweather_api_key" | base64

```


Copy the long random alphanumeric string output.
2. Create a file called `k8s/secret.yaml`:
```bash
nano k8s/secret.yaml

```


3. Paste this structure, replacing `PASTE_YOUR_BASE64_STRING_HERE` with the string you just copied:

```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: weather-secrets
   type: Opaque
   data:
     API_KEY: PASTE_YOUR_BASE64_STRING_HERE

```

### 2. The Redis Cache Layer (`k8s/redis.yaml`)

To support horizontal scaling, we deploy a standalone container using the official `redis:7-alpine` image paired with a stable internal cluster `Service` named `redis-service` targeting internal port `6379`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: weather-redis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: weather-redis
  template:
    metadata:
      labels:
        app: weather-redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis-service
spec:
  selector:
    app: weather-redis
  ports:
    - protocol: TCP
      port: 6379
      targetPort: 6379

```

### 3. The Backend Manifest (`k8s/backend.yaml`)

This deployment handles our Go binary container. It links to the secret value above to populate `WEATHER_API_KEY`, injects the `REDIS_ADDR` pointing to our internal `redis-service:6379`, and defines a `NodePort` service mapping port `8080` out via node port `30080`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: weather-backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: weather-backend
  template:
    metadata:
      labels:
        app: weather-backend
    spec:
      containers:
      - name: backend
        image: weather-backend:v1
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
        env:
        - name: WEATHER_API_KEY
          valueFrom:
            secretKeyRef:
              name: weather-secrets
              key: API_KEY
        - name: REDIS_ADDR
          value: "redis-service:6379"
---
apiVersion: v1
kind: Service
metadata:
  name: backend-service
spec:
  type: NodePort
  selector:
    app: weather-backend
  ports:
    - protocol: TCP
      port: 8080
      targetPort: 8080
      nodePort: 30080

```

### 4. The Frontend Manifest (`k8s/frontend.yaml`)

For the frontend, the service configuration will use `nodePort: 30000`, which hooks straight into host port `4200` that we opened in our initial Kind setup.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: weather-frontend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: weather-frontend
  template:
    metadata:
      labels:
        app: weather-frontend
    spec:
      containers:
      - name: frontend
        image: weather-frontend:v1
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: frontend-service
spec:
  type: NodePort
  selector:
    app: weather-frontend
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
      nodePort: 30000

```

---

## 🧭 Step 4: Deploying to the Cluster

We use `kubectl apply` to submit these structural layers to Kubernetes. Run these commands sequentially from your root project directory:

```bash
# 1. Apply the Secret first so the backend can consume it safely
kubectl apply -f k8s/secret.yaml

# 2. Spin up the Redis infrastructure database
kubectl apply -f k8s/redis.yaml

# 3. Apply the Backend API resources
kubectl apply -f k8s/backend.yaml

# 4. Apply the Frontend web interface resources
kubectl apply -f k8s/frontend.yaml

```

### 🧪 Verifying the Deployment

Monitor your resources in real-time until every single container instance moves to a status of `Running`:

```bash
kubectl get pods -w

```

Check your port maps and routing definitions:

```bash
kubectl get services

```
To scale up (e.g. spin 3 backend pods):
```bash
kubectl scale deployment weather-backend --replicas=3
```
All backend pods read/write to single redis instance:
```bash
kubectl get pods
NAME                               READY   STATUS    RESTARTS      AGE
weather-backend-588b9ff47-7tc84    1/1     Running   4 (24h ago)   2d5h
weather-backend-588b9ff47-mb8sn    1/1     Running   4 (24h ago)   2d5h
weather-backend-588b9ff47-ztxnz    1/1     Running   4 (24h ago)   2d5h
weather-frontend-9b59849ff-8nlts   1/1     Running   4 (24h ago)   2d5h
weather-redis-6cb4b48c94-4hnw4     1/1     Running   4 (24h ago)   2d5h
```

Now, open your Windows browser and head to `http://localhost:4200` to interact with your scalable, distributed full-stack application!
