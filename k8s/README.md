# Kubernetes Deployment Guide

This directory contains Kubernetes manifests for deploying the Ethereum Validator Monitor in a production environment.

## Prerequisites

- Kubernetes cluster (1.24+)
- `kubectl` configured to access your cluster
- Container registry with the validator-monitor image
- Storage class configured for PersistentVolumeClaims
- LoadBalancer support (or Ingress controller for alternative exposure)

## Quick Start

### 1. Update Configuration

Before deploying, update the following:

**secrets.yaml:**
```bash
# Replace placeholder secrets with actual values
vi k8s/secret.yaml
# Update:
# - DB_PASSWORD
# - BEACON_NODE_URL
# - EXECUTION_NODE_URL
# - grafana-admin password
```

**deployment-validator-monitor.yaml:**
```bash
# Update container image registry
vi k8s/deployment-validator-monitor.yaml
# Change: your-registry/validator-monitor:latest
# To: your-actual-registry.io/validator-monitor:v1.0.0
```

**pvc.yaml:**
```bash
# Update storage class if needed (default: standard)
vi k8s/pvc.yaml
# Change storageClassName to match your cluster
```

### 2. Deploy to Kubernetes

```bash
# Deploy in order (respecting dependencies)

# 1. Create namespace
kubectl apply -f k8s/namespace.yaml

# 2. Create ConfigMaps and Secrets
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml

# 3. Create PersistentVolumeClaims
kubectl apply -f k8s/pvc.yaml

# 4. Deploy infrastructure services
kubectl apply -f k8s/deployment-postgres.yaml
kubectl apply -f k8s/deployment-redis.yaml

# 5. Wait for infrastructure to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n validator-monitor --timeout=120s
kubectl wait --for=condition=ready pod -l app=redis -n validator-monitor --timeout=120s

# 6. Deploy monitoring stack
kubectl apply -f k8s/deployment-prometheus.yaml
kubectl apply -f k8s/deployment-grafana.yaml

# 7. Deploy application
kubectl apply -f k8s/deployment-validator-monitor.yaml

# 8. Create services
kubectl apply -f k8s/service.yaml
```

**Or deploy all at once:**
```bash
kubectl apply -f k8s/
```

### 3. Verify Deployment

```bash
# Check all resources
kubectl get all -n validator-monitor

# Check pod status
kubectl get pods -n validator-monitor -w

# Check logs
kubectl logs -f deployment/validator-monitor -n validator-monitor

# Check persistent volumes
kubectl get pvc -n validator-monitor
```

### 4. Access Services

**Get LoadBalancer IPs:**
```bash
# Validator Monitor API
kubectl get svc validator-monitor -n validator-monitor
# Access at http://<EXTERNAL-IP>:8080

# Grafana
kubectl get svc grafana -n validator-monitor
# Access at http://<EXTERNAL-IP>:3000

# Prometheus (internal only by default)
kubectl port-forward svc/prometheus 9090:9090 -n validator-monitor
# Access at http://localhost:9090
```

## Architecture

```
┌─────────────────────────────────────────────────┐
│            Kubernetes Cluster                   │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │  Namespace: validator-monitor            │  │
│  │                                          │  │
│  │  ┌────────────┐  ┌─────────────┐        │  │
│  │  │  Grafana   │  │ Prometheus  │        │  │
│  │  │ (1 replica)│◄─┤ (1 replica) │        │  │
│  │  └─────┬──────┘  └──────▲──────┘        │  │
│  │        │                 │               │  │
│  │        │      ┌──────────┴────────┐     │  │
│  │        │      │ Validator Monitor │     │  │
│  │        │      │   (2 replicas)    │     │  │
│  │        │      └─────────┬─────────┘     │  │
│  │        │                │               │  │
│  │        │      ┌─────────┴────────┐      │  │
│  │        │      │                  │      │  │
│  │        │  ┌───▼────┐      ┌─────▼───┐  │  │
│  │        │  │Postgres│      │  Redis  │  │  │
│  │        │  │   DB   │      │  Cache  │  │  │
│  │        │  └────────┘      └─────────┘  │  │
│  │        │                                │  │
│  │        │  ┌────────────────────────┐   │  │
│  │        └──► PersistentVolumes (4)  │   │  │
│  │           └────────────────────────┘   │  │
│  └──────────────────────────────────────┘  │
│                                             │
│  ┌──────────────┐  ┌──────────────┐        │
│  │ LoadBalancer │  │ LoadBalancer │        │
│  │  (Grafana)   │  │ (Val Monitor)│        │
│  └──────┬───────┘  └──────┬───────┘        │
└─────────┼──────────────────┼────────────────┘
          │                  │
          ▼                  ▼
   Users/Dashboards      API Clients
```

## Configuration Management

### ConfigMaps

- `validator-monitor-config` - Application configuration
- `prometheus-config` - Prometheus scrape and evaluation config
- `prometheus-alerts` - Alert rules (subset, full rules should be mounted)
- `grafana-datasource` - Prometheus datasource config

### Secrets

- `validator-monitor-secrets` - DB credentials, node URLs
- `grafana-admin` - Grafana admin credentials

**Best practices:**
- Use [sealed-secrets](https://github.com/bitnami-labs/sealed-secrets) for GitOps
- Or use external-secrets with AWS Secrets Manager, HashiCorp Vault, etc.

## Resource Sizing

**Development/Testing:**
- postgres: 256Mi/250m CPU
- redis: 128Mi/100m CPU
- prometheus: 512Mi/250m CPU
- grafana: 256Mi/200m CPU
- validator-monitor: 512Mi/500m CPU

**Production (recommended starting point):**
- postgres: 2Gi/1 CPU
- redis: 512Mi/250m CPU
- prometheus: 4Gi/1 CPU (depends on retention)
- grafana: 512Mi/500m CPU
- validator-monitor: 2Gi/2 CPU (scale horizontally)

## High Availability

### Database HA

For production, consider:
- [PostgreSQL Operator](https://github.com/zalando/postgres-operator)
- [Crunchy PostgreSQL Operator](https://github.com/CrunchyData/postgres-operator)
- Managed database (AWS RDS, GCP Cloud SQL, etc.)

### Redis HA

For production, consider:
- Redis Sentinel deployment
- Redis Cluster
- Managed Redis (AWS ElastiCache, GCP Memorystore, etc.)

### Application Scaling

```bash
# Scale validator-monitor horizontally
kubectl scale deployment validator-monitor -n validator-monitor --replicas=4

# Autoscaling based on metrics
kubectl autoscale deployment validator-monitor \
  -n validator-monitor \
  --cpu-percent=70 \
  --min=2 \
  --max=10
```

## Monitoring & Observability

### Prometheus Metrics

Metrics exposed by validator-monitor at `:9090/metrics`:
- Application performance metrics
- Validator effectiveness scores
- API latency and error rates
- Database connection pool status
- Cache hit/miss rates

### Grafana Dashboards

Pre-configured dashboard includes:
- Validator health overview
- Performance trends
- System health metrics
- Alert status

### Logs

```bash
# View application logs
kubectl logs -f deployment/validator-monitor -n validator-monitor

# View logs from all replicas
kubectl logs -f deployment/validator-monitor -n validator-monitor --all-containers=true

# Stream logs from specific pod
kubectl logs -f <pod-name> -n validator-monitor
```

## Backup & Recovery

### Database Backups

```bash
# Manual backup
kubectl exec -it deployment/postgres -n validator-monitor -- \
  pg_dump -U postgres validator_monitor > backup-$(date +%Y%m%d).sql

# Restore from backup
kubectl exec -i deployment/postgres -n validator-monitor -- \
  psql -U postgres validator_monitor < backup-20250118.sql
```

### Persistent Volume Snapshots

```bash
# Create VolumeSnapshot (if supported by storage class)
kubectl create -f - <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: postgres-snapshot-$(date +%Y%m%d)
  namespace: validator-monitor
spec:
  volumeSnapshotClassName: standard-snapshot
  source:
    persistentVolumeClaimName: postgres-pvc
EOF
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod events
kubectl describe pod <pod-name> -n validator-monitor

# Check resource constraints
kubectl top pods -n validator-monitor

# Check if PVC is bound
kubectl get pvc -n validator-monitor
```

### Connection Issues

```bash
# Test database connectivity
kubectl exec -it deployment/validator-monitor -n validator-monitor -- \
  nc -zv postgres 5432

# Test Redis connectivity
kubectl exec -it deployment/validator-monitor -n validator-monitor -- \
  nc -zv redis 6379

# Check DNS resolution
kubectl exec -it deployment/validator-monitor -n validator-monitor -- \
  nslookup postgres.validator-monitor.svc.cluster.local
```

### Performance Issues

```bash
# Check resource usage
kubectl top pods -n validator-monitor

# Check for OOM kills
kubectl describe pod <pod-name> -n validator-monitor | grep -i oom

# View Prometheus metrics
kubectl port-forward svc/prometheus 9090:9090 -n validator-monitor
# Open http://localhost:9090
```

## Security Considerations

1. **Non-root containers**: All containers run as non-root users
2. **Read-only root filesystem**: Validator monitor uses read-only root FS
3. **Resource limits**: All containers have memory and CPU limits
4. **Network policies**: Consider adding NetworkPolicies for pod-to-pod communication
5. **Secrets management**: Use external secrets manager in production
6. **RBAC**: Follow principle of least privilege for service accounts

## Cleanup

```bash
# Delete all resources
kubectl delete namespace validator-monitor

# This will delete:
# - All pods and deployments
# - All services
# - All configmaps and secrets
# - All PVCs (and associated PVs)
```

## Next Steps

- Set up Ingress for HTTPS access
- Configure HorizontalPodAutoscaler for auto-scaling
- Set up monitoring alerts (Alertmanager)
- Configure log aggregation (ELK, Loki)
- Implement GitOps with ArgoCD or Flux
- Set up CI/CD pipeline for automated deployments
