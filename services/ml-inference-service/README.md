# ML 추론 서비스 (ML Inference Service)

**ML Inference Service**는 딥러닝 모델을 프로덕션 환경에서 고성능으로 서빙하기 위한 인프라입니다. **NVIDIA Triton Inference Server**를 기반으로 구축되었습니다.

## 🛠 주요 기능 (Features)

### 1. 고성능 모델 서빙
-   **다중 프레임워크 지원**: PyTorch, TensorFlow, ONNX 등 다양한 포맷의 모델을 단일 서버에서 서빙합니다.
-   **TensorRT 최적화**: NVIDIA GPU에 최적화된 TensorRT 엔진을 사용하여 추론 속도를 극대화합니다.

### 2. 동적 배칭 (Dynamic Batching)
-   **요청 병합**: 짧은 시간 내에 들어오는 개별 추론 요청들을 서버 내부에서 하나의 배치(Batch)로 묶어 처리합니다.
-   **처리량(Throughput) 향상**: 개별 요청의 응답 속도를 크게 해치지 않으면서 전체 처리량을 비약적으로 높입니다.

### 3. 앙상블 파이프라인 (Ensemble Pipeline)
-   **전처리/후처리 통합**: 데이터 전처리(Python), 모델 추론(TensorRT), 후처리(Python) 단계를 하나의 파이프라인으로 구성하여 네트워크 오버헤드를 줄입니다.

## 📂 모델 저장소 구조 (Model Repository)
```
model_repository/
└── price_prediction_lstm/
    ├── config.pbtxt      # 모델 설정 (입출력, 배치 크기 등)
    └── 1/
        └── model.pt      # 학습된 모델 파일
```

## 🚀 시작하기 (Getting Started)

### Docker 실행
```bash
docker run --gpus all --rm -p 8000:8000 -p 8001:8001 -p 8002:8002 \
  -v $(pwd)/model_repository:/models \
  nvcr.io/nvidia/tritonserver:23.10-py3 \
  tritonserver --model-repository=/models
```
