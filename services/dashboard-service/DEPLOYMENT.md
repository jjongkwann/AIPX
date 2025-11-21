# Deployment Guide

Dashboard Service 프로덕션 배포 가이드입니다.

## 배포 옵션

1. Docker + Docker Compose
2. Kubernetes
3. Vercel (권장)
4. AWS / GCP / Azure

## 1. Docker 배포

### 빌드

```bash
# 이미지 빌드
docker build -t aipx-dashboard:latest .

# 이미지 확인
docker images | grep aipx-dashboard
```

### 실행

```bash
# 단일 컨테이너 실행
docker run -d \
  -p 3000:3000 \
  -e NEXT_PUBLIC_API_BASE=https://api.aipx.com \
  -e NEXT_PUBLIC_WS_BASE=wss://ws.aipx.com \
  --name aipx-dashboard \
  aipx-dashboard:latest

# 로그 확인
docker logs -f aipx-dashboard
```

### Docker Compose

```bash
# 전체 스택 실행
docker-compose up -d

# 로그 확인
docker-compose logs -f dashboard

# 종료
docker-compose down
```

## 2. Vercel 배포 (권장)

### Vercel CLI 설치

```bash
npm install -g vercel
```

### 배포

```bash
# 로그인
vercel login

# 프로덕션 배포
vercel --prod

# 프리뷰 배포
vercel
```

### 환경 변수 설정

Vercel 대시보드에서 설정:

1. Project Settings > Environment Variables
2. 다음 변수 추가:
   - `NEXT_PUBLIC_API_BASE`
   - `NEXT_PUBLIC_WS_BASE`
   - `NEXTAUTH_URL`
   - `NEXTAUTH_SECRET`

### vercel.json 설정

```json
{
  "framework": "nextjs",
  "buildCommand": "npm run build",
  "devCommand": "npm run dev",
  "installCommand": "npm install",
  "regions": ["icn1"],
  "env": {
    "NEXT_PUBLIC_API_BASE": "@api-base-url",
    "NEXT_PUBLIC_WS_BASE": "@ws-base-url"
  }
}
```

## 3. Kubernetes 배포

### Deployment 생성

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aipx-dashboard
  labels:
    app: dashboard
spec:
  replicas: 3
  selector:
    matchLabels:
      app: dashboard
  template:
    metadata:
      labels:
        app: dashboard
    spec:
      containers:
      - name: dashboard
        image: aipx-dashboard:latest
        ports:
        - containerPort: 3000
        env:
        - name: NEXT_PUBLIC_API_BASE
          value: "https://api.aipx.com"
        - name: NEXT_PUBLIC_WS_BASE
          value: "wss://ws.aipx.com"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5
```

### Service 생성

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: dashboard-service
spec:
  selector:
    app: dashboard
  ports:
  - protocol: TCP
    port: 80
    targetPort: 3000
  type: LoadBalancer
```

### Ingress 생성

```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: dashboard-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - aipx.com
    secretName: aipx-tls
  rules:
  - host: aipx.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: dashboard-service
            port:
              number: 80
```

### 배포 실행

```bash
# 배포 적용
kubectl apply -f k8s/

# 상태 확인
kubectl get pods -l app=dashboard
kubectl get svc dashboard-service
kubectl get ingress dashboard-ingress

# 로그 확인
kubectl logs -f deployment/aipx-dashboard
```

## 4. AWS 배포

### ECR에 이미지 푸시

```bash
# ECR 로그인
aws ecr get-login-password --region ap-northeast-2 | \
  docker login --username AWS --password-stdin <account-id>.dkr.ecr.ap-northeast-2.amazonaws.com

# 이미지 태그
docker tag aipx-dashboard:latest <account-id>.dkr.ecr.ap-northeast-2.amazonaws.com/aipx-dashboard:latest

# 이미지 푸시
docker push <account-id>.dkr.ecr.ap-northeast-2.amazonaws.com/aipx-dashboard:latest
```

### ECS 배포

1. ECS 클러스터 생성
2. Task Definition 생성
3. Service 생성
4. Load Balancer 설정

## 환경 변수 관리

### Production 환경 변수

```env
NODE_ENV=production
NEXT_PUBLIC_API_BASE=https://api.aipx.com
NEXT_PUBLIC_WS_BASE=wss://ws.aipx.com
NEXTAUTH_URL=https://aipx.com
NEXTAUTH_SECRET=<strong-random-secret>
```

### 시크릿 관리

**AWS Secrets Manager**:

```bash
aws secretsmanager create-secret \
  --name aipx-dashboard-secrets \
  --secret-string file://secrets.json
```

**Kubernetes Secrets**:

```bash
kubectl create secret generic dashboard-secrets \
  --from-literal=nextauth-secret=<secret> \
  --from-literal=api-key=<api-key>
```

## SSL/TLS 설정

### Let's Encrypt (Certbot)

```bash
# Certbot 설치
sudo apt-get install certbot python3-certbot-nginx

# 인증서 발급
sudo certbot --nginx -d aipx.com -d www.aipx.com

# 자동 갱신 설정
sudo certbot renew --dry-run
```

### Nginx SSL 설정

```nginx
server {
    listen 443 ssl http2;
    server_name aipx.com;

    ssl_certificate /etc/letsencrypt/live/aipx.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/aipx.com/privkey.pem;

    # SSL 최적화
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
}
```

## 모니터링

### Health Check

```bash
# 헬스 체크 엔드포인트
curl https://aipx.com/api/health

# 응답 예시
{
  "status": "healthy",
  "timestamp": "2025-01-21T10:00:00Z"
}
```

### 로그 수집

**CloudWatch (AWS)**:

```javascript
// next.config.js
module.exports = {
  logging: {
    fetches: {
      fullUrl: true,
    },
  },
}
```

**Datadog**:

```bash
# Datadog Agent 설치
DD_API_KEY=<api-key> DD_SITE="datadoghq.com" bash -c \
  "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"
```

## 성능 최적화

### CDN 설정

**CloudFlare**:
1. DNS 레코드 추가
2. Auto Minify 활성화
3. Brotli 압축 활성화
4. 캐시 설정

### 이미지 최적화

```javascript
// next.config.js
module.exports = {
  images: {
    domains: ['cdn.aipx.com'],
    formats: ['image/webp', 'image/avif'],
    deviceSizes: [640, 750, 828, 1080, 1200, 1920],
  },
}
```

## 롤백 전략

### Docker

```bash
# 이전 버전으로 롤백
docker stop aipx-dashboard
docker rm aipx-dashboard
docker run -d --name aipx-dashboard aipx-dashboard:previous
```

### Kubernetes

```bash
# 배포 히스토리 확인
kubectl rollout history deployment/aipx-dashboard

# 이전 버전으로 롤백
kubectl rollout undo deployment/aipx-dashboard

# 특정 리비전으로 롤백
kubectl rollout undo deployment/aipx-dashboard --to-revision=2
```

### Vercel

```bash
# 배포 목록 확인
vercel ls

# 특정 배포로 롤백
vercel promote <deployment-url>
```

## CI/CD 파이프라인

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'

      - name: Install dependencies
        run: npm ci

      - name: Build
        run: npm run build

      - name: Deploy to Vercel
        uses: amondnet/vercel-action@v25
        with:
          vercel-token: ${{ secrets.VERCEL_TOKEN }}
          vercel-org-id: ${{ secrets.ORG_ID }}
          vercel-project-id: ${{ secrets.PROJECT_ID }}
          vercel-args: '--prod'
```

## 보안 체크리스트

- [ ] HTTPS 강제
- [ ] 환경 변수 암호화
- [ ] CORS 설정
- [ ] Rate limiting
- [ ] 보안 헤더 설정
- [ ] 의존성 취약점 점검
- [ ] 정기적인 보안 업데이트

## 트러블슈팅

### 빌드 실패

```bash
# 캐시 삭제
rm -rf .next node_modules
npm install
npm run build
```

### 메모리 부족

```bash
# Node.js 메모리 제한 증가
NODE_OPTIONS=--max-old-space-size=4096 npm run build
```

### 연결 타임아웃

nginx 설정에서 타임아웃 증가:

```nginx
proxy_connect_timeout 600;
proxy_send_timeout 600;
proxy_read_timeout 600;
```

## 백업 및 복구

### 설정 백업

```bash
# 환경 변수 백업
cp .env.production .env.production.backup

# Docker 이미지 백업
docker save aipx-dashboard:latest | gzip > dashboard-backup.tar.gz
```

### 복구

```bash
# 이미지 복구
docker load < dashboard-backup.tar.gz
```

## 추가 리소스

- [Next.js Deployment](https://nextjs.org/docs/deployment)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Vercel Documentation](https://vercel.com/docs)
