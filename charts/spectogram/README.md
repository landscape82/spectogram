# spectogram Helm chart

Deploys the Spectogram web viewer (Deployment + Service, plus optional
Ingress and a one-off "generate" Job) using the image built from the
repo root [`Dockerfile`](../../Dockerfile).

The chart is written so that moving from a local `kind` cluster to a
cloud or on-prem cluster is purely a matter of `--set`/values overrides
— no template changes are needed.

## Quickstart on `kind`

```bash
# from the repo root
docker build -t spectogram:local .
kind create cluster --name spectogram
kind load docker-image spectogram:local --name spectogram

helm install spectogram charts/spectogram \
  --set image.repository=spectogram \
  --set image.tag=local \
  --wait

helm test spectogram

kubectl port-forward svc/spectogram 8000:8000
# open http://localhost:8000/web
```

There's no spectrogram data yet at this point (see "Generating data"
below).

## Generating data

The chart can run the Go CLI's `generate` step as a one-off Job before
the viewer starts, sharing the same PersistentVolumeClaim as the
`serve` Deployment:

```bash
kubectl create configmap my-audio --from-file=audio.wav=./audio.wav

helm upgrade spectogram charts/spectogram \
  --set image.repository=spectogram \
  --set image.tag=local \
  --set generateJob.enabled=true \
  --set generateJob.audioConfigMap=my-audio \
  --set generateJob.audioKey=audio.wav \
  --set generateJob.inputPath=/app/audio-input/audio.wav \
  --wait
```

`generateJob.audioKey` and the trailing path segment of
`generateJob.inputPath` must match (the ConfigMap is mounted at
`/app/audio-input`, so key `audio.wav` appears at
`/app/audio-input/audio.wav`). The audio format is detected from the
file extension (`.mp3` or `.wav`), so the key/path extension must match
the actual file content.

Alternatively, `kubectl cp` an audio file into a running pod and run
`spectogram generate` manually, or `kubectl cp` pre-generated
`spectrogram.png`/`data/spectrogram.json` directly onto the volume.

## Deploying to a cloud or on-prem cluster

The defaults (`ClusterIP` Service, chart-managed PVC using the
cluster's default StorageClass, no Ingress) work unmodified on `kind`.
For a real cluster:

```bash
helm install spectogram charts/spectogram \
  --set image.repository=ghcr.io/landscape82/spectogram \
  --set image.tag=v0.1.0 \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.host=spectogram.example.com \
  --set persistence.storageClassName=<your-storage-class> \
  --set resources.requests.cpu=50m \
  --set resources.requests.memory=64Mi \
  --set resources.limits.cpu=250m \
  --set resources.limits.memory=256Mi
```

- **Image**: push the image built from the repo `Dockerfile` to your
  registry and set `image.repository`/`image.tag` (and
  `imagePullSecrets` if the registry is private).
- **Exposure**: set `service.type=LoadBalancer` on a cloud provider
  that supports it, `service.type=NodePort` for bare on-prem clusters
  without a load balancer, or enable `ingress` with your ingress
  controller's class.
- **Storage**: set `persistence.storageClassName` to a StorageClass
  available in your cluster, or set `persistence.existingClaim` to
  reuse an existing PVC. Leave it empty to use the cluster's default
  StorageClass (this is what makes the chart work unmodified on
  `kind`, which registers a default StorageClass out of the box).
- **Scaling**: enable `autoscaling.enabled=true` if your cluster has
  the metrics-server needed for CPU-based HPA scaling. Note the
  `generateJob`/PVC model here assumes `ReadWriteOnce`, single-writer
  access — it's not designed for multi-replica concurrent writes.

## Values reference

See [`values.yaml`](values.yaml) for the full list of configurable
values and inline documentation for each one.

## Testing

- `helm lint charts/spectogram`
- `helm template spectogram charts/spectogram` to review rendered
  manifests
- `helm install ... --wait && helm test spectogram` runs a `helm test`
  hook Pod that fetches `/web/index.html` from the Service and asserts
  the Plotly viewer page is served

CI (`.github/workflows/helm.yml`) lints and templates the chart on
every PR touching it, and additionally spins up a real `kind` cluster
to `helm install` and `helm test` the chart end-to-end.
