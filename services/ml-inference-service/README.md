# ML Inference Service

Real-time ML model inference service using NVIDIA Triton Inference Server for AIPX trading system.

## Overview

The ML Inference Service provides AI-powered market predictions by serving:
- **LSTM Price Predictor**: Next-tick price prediction using LSTM neural networks
- **Transformer Sentiment Analyzer**: News sentiment analysis using transformer models
- **Ensemble Predictor**: Combined predictions leveraging both models

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    ML Inference Service                      │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐         ┌─────────────────────────┐       │
│  │   FastAPI    │◄────────┤  Strategy Services      │       │
│  │   Gateway    │         └─────────────────────────┘       │
│  └──────┬───────┘                                            │
│         │                                                     │
│         ▼                                                     │
│  ┌──────────────────────────────────────────────────┐       │
│  │        NVIDIA Triton Inference Server            │       │
│  ├──────────────────────────────────────────────────┤       │
│  │  ┌────────────┐  ┌─────────────┐  ┌───────────┐ │       │
│  │  │   LSTM     │  │ Transformer │  │ Ensemble  │ │       │
│  │  │   Model    │  │   Sentiment │  │  Combiner │ │       │
│  │  └────────────┘  └─────────────┘  └───────────┘ │       │
│  └──────────────────────────────────────────────────┘       │
│                                                               │
│  ┌──────────────────────────────────────────────────┐       │
│  │         Feature Engineering Pipeline             │       │
│  └──────────────────────────────────────────────────┘       │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Features

### 1. LSTM Price Prediction
- Input: 60 timesteps of OHLCV data
- Output: Next tick price prediction
- Features:
  - Attention mechanism for interpretability
  - Batch processing support
  - Dynamic batching for optimal throughput
  - GPU acceleration

### 2. Sentiment Analysis
- Input: News text (up to 512 tokens)
- Output: Sentiment score (-1 to +1) and confidence
- Features:
  - Transformer-based architecture
  - ONNX runtime optimization
  - TensorRT acceleration
  - Multi-language support

### 3. Ensemble Predictions
- Combines price and sentiment signals
- Weighted averaging with confidence scores
- Triton ensemble pipeline
- Single inference call

### 4. Feature Engineering
- Automated feature extraction from OHLCV data
- Technical indicators (RSI, MA, volatility)
- Real-time normalization
- Batch preprocessing

### 5. Model Monitoring
- Inference latency tracking (p50, p95, p99)
- Error rate monitoring
- Model drift detection
- Performance alerts
- Prometheus metrics export

## Quick Start

### Prerequisites
- Docker with NVIDIA GPU support
- NVIDIA Driver >= 525.60.13
- CUDA >= 12.0
- Python 3.11+

### Installation

1. Clone repository:
```bash
git clone <repository-url>
cd services/ml-inference-service
```

2. Set up environment:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Start services:
```bash
docker-compose up -d
```

4. Verify health:
```bash
curl http://localhost:8005/health
```
