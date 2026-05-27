# 🌤️ Weather App (K8s + Redis)

A scalable, production-grade full-stack weather application featuring an **Angular** frontend and a stateless **Go** backend.\
The application fetches real-time meteorological data from the OpenWeather API and utilizes a centralized, distributed **Redis** caching layer to handle rapid repeated requests efficiently.\
The entire ecosystem is orchestrated natively inside a local **Kubernetes (Kind)** cluster, moving away from simple single-host container runtimes to modern cloud-native standards.

<img width="362" height="491" alt="image" src="https://github.com/user-attachments/assets/3c39336d-36f8-46bd-b941-ee5d4dd45284" />

## 🏗️ System Architecture

Unlike traditional monolithic setups, this architecture isolates compute and caching into dedicated, decoupled components:

*   **Frontend (Angular + Nginx):** A containerized single-page web interface served via an optimized Nginx web server, exposed to the host machine through a native Kubernetes NodePort Service.
*   **Backend (Go REST API):** A stateless Go microservice that handles routing, business logic, and third-party API communication. It automatically pulls api_key from Kubernetes Secrets.
*   **Cache Layer (Redis):** A centralized `redis:7-alpine` database instance. Because the backend instances are fully stateless, multiple replicas can scale horizontally while communicating with this shared cache to eliminate calls to OpenWeather.Redis cache implementation handling automatic TTL (Time-To-Live) expirations.\
We support a Stale-While-Revalidate (SWR) data pipeline to eliminate duplicate API latency by serving stale data and fetching asynchronously new data (via go routine).
*   **Infrastructure (Kind):** A local Kubernetes cluster executing via Docker containers, utilizing internal cluster networking DNS (`redis-service:6379`) for secure intra-component communication.
*   **Horizontal Pod Scalability:** Besides manual Kubernetes replication scaling (`kubectl scale`), we have a **Horizontal Pod Autoscaler (HPA)**  that monitors CPU utilization. Because the Go backend is stateless and speaks to a shared Redis layer, Kubernetes can dynamically scale backend instances from **1 to 5 replicas** on the fly to absorb heavy traffic bursts.
*   **Kubernetes Metrics Server** A local monitoring tool that measures the real-time CPU and memory load of our pods, allowing the cluster to make automated scaling decisions.
*   **Prometheus & Grafana Monitoring** A telemetry pipeline where Prometheus automatically scrapes system metrics from the Go backend (every 5 seconds), and Grafana attaches to this data to provide dashboards tracking traffic spikes and cache hit/miss ratios.
---

## 🚀 Getting Started & Local Deployment

To abstract heavy technical configurations away from product documentation, all prerequisites, cluster initialization, container building, metric server setup, prometheus/grafana setup, load testing and Kubernetes manifest steps are detailed in a dedicated setup guide.

### 📖 [Click to view the step-by-step Local K8s Setup Guide](./SETUP.md)

---

## 🛠️ Tech Stack

*   **Frontend:** Angular 17+, Nginx
*   **Backend:** Go (Golang) 1.22+, `go-redis/v9`
*   **Database/Cache:** Redis 7
*   **Orchestration:** Kubernetes v1.29+, Kind (Kubernetes in Docker), `kubectl`
*   **Metrics Instrumentation:** Official Prometheus Go Client Library (`client_golang`)
*   **Data Visualization:** Grafana v10.0+
*   **Traffic Simulation:** ApacheBench (`ab`) for load-testing

## ⚙️ Getting Started
🐳 with K8S

start the cluster:
```bash
docker start weather-cluster-control-plane

kubectl scale deployment weather-backend --replicas=3
kubectl scale deployment weather-frontend --replicas=1
kubectl scale deployment weather-redis --replicas=1
```
The frontend will start on:
http://localhost:4200.

🐳 Without K8S : Containerization (Docker)

If you prefer NOT installing kubectl or kind, I’ve included an optional `docker-compose.yml` that containerizes the frontend, backend, and Redis.\
So, you can run the entire stack via Docker Compose. This automatically builds the images and spins up the Go, Angular, and Redis containers.\
Launch the containers
```bash
docker compose up --build
```
To ensure both the containers are running, run:
```bash
docker compose ps
NAMES STATUS
go-weather-grafana-1                      Up 20 seconds
go-weather-prometheus                     Up 20 seconds
go-weather-redis-1                        Up 20 seconds
go-weather-frontend-1                     Up 20 seconds
go-weather-backend-1                      Up 21 seconds
```
Once started, access: http://localhost:4200

## 🔄 How It Works
Users can retrieve weather data by searching for a specific city or by using the browser's geolocation to fetch data for their current location.\
Frontend sends a GET request: `http://localhost:8080/weather?city=tokyo` to backend.

Go Backend & Cache Pipeline:

- Receives incoming requests from the frontend 
- queries the Redis cache layer via a non-blocking Stale-While-Revalidate (SWR) check:
  - Cache Hit (Fresh): If cached data exists within the 1-minute freshness window, it returns the response immediately (X-Cache: FROM_CACHE).
  - Cache Hit (Stale): If the data is found but has passed the 1-minute freshness window, the backend instantly serves the stale data to the client (X-Cache: STALE_REVALIDATING) while asynchronously spawning a detached Goroutine to fetch fresh data from the OpenWeather API and update Redis in the background.
  - Cache Miss: If the data is completely absent from Redis, it performs a synchronous fallback call to the OpenWeather API using the securely injected Kubernetes Secret key, structures the payload into a simplified JSON layout, commits it to Redis with an extended safety TTL(5 min), and returns the response directly to the frontend.
- Telemetry: During every cache lookup, the backend uses the official Go Prometheus client to increment counters (weather_cache_hits_total or weather_cache_misses_total), keeping track of application performance for real-time scraping.

Frontend Rendering: Angular receives the JSON response and renders
  - Temperature (°C / °F)
  - Humidity
  - Wind data
  - Weather condition and icon

Backend showcase request with and w/o caching
<img width="1681" height="111" alt="image" src="https://github.com/user-attachments/assets/3024f2d3-7021-42ff-8ee6-84d7fe88758c" />

## 🧪 Testing
Run Angular tests (Karma/Jasmine)

```bash
frontend\weather-app> ng test
```

## 📁 Structure
weatherApp_Angular_and_Go (GitHub Repo)
```bash
├── backend/
│ ├── Dockerfile
│ ├── models.go
│ ├── weather.go
│ └── weather_service.go
├── frontend/
│ ├── Dockerfile
│ └── weather-app/
│ ├── src/
│ └── package.json
└── .gitignore
├── docker-compose.yml
├── prometheus.yml
├── kind-config.yaml
├── k8s/
│ ├── hpa.yaml
│ ├── monitoring.yaml
│ ├── backend.yaml
│ ├── frontend.yaml
│ ├── redist.yaml
│ └── secret.yaml
└── README.md
└── SETUP.md
```
