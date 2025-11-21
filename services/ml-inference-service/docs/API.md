# API Documentation

Complete API reference for ML Inference Service.

## Base URL

```
http://localhost:8005/api/v1
```

## Authentication

Currently no authentication required. In production, use JWT tokens or API keys.

## Endpoints

### 1. Health Check

Check service health and available models.

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy",
  "triton_connected": true,
  "available_models": [
    "lstm_price_predictor",
    "transformer_sentiment",
    "ensemble_predictor"
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Status Codes**:
- `200`: Service healthy
- `503`: Service unhealthy

---

### 2. Price Prediction

Predict next tick price using LSTM model.

**Endpoint**: `POST /api/v1/predict/price`

**Request**:
```json
{
  "symbol": "005930",
  "ohlcv_data": [
    [71000, 71500, 70500, 71200, 1000000],
    [71200, 71800, 71000, 71500, 1200000],
    ...  // 60 rows total
  ],
  "model_version": "1"  // Optional
}
```

**Parameters**:
- `symbol` (string, required): Stock symbol
- `ohlcv_data` (array, required): OHLCV data, shape (60, 5)
  - Each row: [open, high, low, close, volume]
- `model_version` (string, optional): Model version to use

**Response**:
```json
{
  "predicted_price": 71500.0,
  "confidence": 0.85,
  "model_name": "lstm_price_predictor",
  "model_version": "1",
  "inference_time_ms": 12.5,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Status Codes**:
- `200`: Prediction successful
- `400`: Invalid input data
- `500`: Inference error

**Example**:
```python
import requests

response = requests.post(
    'http://localhost:8005/api/v1/predict/price',
    json={
        'symbol': '005930',
        'ohlcv_data': [[...], [...], ...]  # 60 rows
    }
)
print(response.json())
```

---

### 3. Sentiment Analysis

Analyze sentiment of news text.

**Endpoint**: `POST /api/v1/predict/sentiment`

**Request**:
```json
{
  "text": "Samsung reports strong quarterly earnings with record chip sales",
  "model_version": "1"  // Optional
}
```

**Parameters**:
- `text` (string, required): Text to analyze (max 512 tokens)
- `model_version` (string, optional): Model version to use

**Response**:
```json
{
  "sentiment": 0.78,
  "confidence": 0.92,
  "model_name": "transformer_sentiment",
  "model_version": "1",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Sentiment Values**:
- `-1.0 to -0.3`: Strong negative
- `-0.3 to 0.0`: Negative
- `0.0 to 0.3`: Neutral
- `0.3 to 0.7`: Positive
- `0.7 to 1.0`: Strong positive

**Status Codes**:
- `200`: Analysis successful
- `400`: Invalid input
- `500`: Inference error

---

### 4. Ensemble Prediction

Combined prediction using both price and sentiment models.

**Endpoint**: `POST /api/v1/predict/ensemble`

**Request**:
```json
{
  "symbol": "005930",
  "ohlcv_data": [[...], [...], ...],  // 60 rows, 5 columns
  "news_text": "Samsung announces new AI chip production line",
  "model_version": "1"  // Optional
}
```

**Parameters**:
- `symbol` (string, required): Stock symbol
- `ohlcv_data` (array, required): OHLCV data, shape (60, 5)
- `news_text` (string, required): Related news text
- `model_version` (string, optional): Model version to use

**Response**:
```json
{
  "final_prediction": 72000.0,
  "price_component": 71500.0,
  "sentiment_component": 0.78,
  "confidence": 0.88,
  "model_name": "ensemble_predictor",
  "model_version": "1",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Status Codes**:
- `200`: Prediction successful
- `400`: Invalid input
- `500`: Inference error

---

### 5. Batch Prediction

Process multiple predictions in a single request.

**Endpoint**: `POST /api/v1/predict/batch`

**Request**:
```json
{
  "items": [
    {
      "symbol": "005930",
      "ohlcv_data": [[...], [...], ...]
    },
    {
      "symbol": "000660",
      "ohlcv_data": [[...], [...], ...]
    }
  ],
  "model_version": "1"
}
```

**Response**:
```json
{
  "predictions": [
    {
      "predicted_price": 71500.0,
      "batch_index": 0,
      "model_version": "1",
      "symbol": "005930"
    },
    {
      "predicted_price": 145000.0,
      "batch_index": 1,
      "model_version": "1",
      "symbol": "000660"
    }
  ],
  "count": 2,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Status Codes**:
- `200`: Batch processing successful
- `400`: Invalid input
- `500`: Inference error

---

### 6. Model Metadata

Get metadata for a specific model.

**Endpoint**: `GET /api/v1/models/{model_name}/metadata`

**Parameters**:
- `model_name` (path): Model name
  - `lstm_price_predictor`
  - `transformer_sentiment`
  - `ensemble_predictor`

**Response**:
```json
{
  "name": "lstm_price_predictor",
  "versions": ["1"],
  "platform": "pytorch_libtorch",
  "inputs": [
    {
      "name": "input__0",
      "datatype": "FP32",
      "shape": [60, 5]
    }
  ],
  "outputs": [
    {
      "name": "output__0",
      "datatype": "FP32",
      "shape": [1]
    }
  ],
  "max_batch_size": 32
}
```

**Status Codes**:
- `200`: Metadata retrieved
- `404`: Model not found

---

### 7. Metrics

Get inference metrics and statistics.

**Endpoint**: `GET /api/v1/metrics`

**Response**:
```json
{
  "server_metrics": {
    "lstm_price_predictor": {
      "count": 1542,
      "total_time_ns": 15420000000,
      "avg_time_ms": 10.0,
      "version": "1"
    }
  },
  "app_metrics": {
    "models": {
      "lstm_price_predictor": {
        "total_requests": 1542,
        "successful_requests": 1540,
        "failed_requests": 2,
        "success_rate": 0.9987,
        "avg_inference_time_ms": 10.2,
        "min_inference_time_ms": 5.1,
        "max_inference_time_ms": 45.3,
        "p50_inference_time_ms": 9.5,
        "p95_inference_time_ms": 15.3,
        "p99_inference_time_ms": 22.1,
        "requests_per_second": 12.5
      }
    },
    "uptime_seconds": 3600,
    "start_time": "2024-01-15T09:30:00Z"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Status Codes**:
- `200`: Metrics retrieved
- `500`: Error retrieving metrics

---

## Error Responses

All errors follow this format:

```json
{
  "detail": "Error message describing what went wrong"
}
```

**Common Error Codes**:
- `400`: Bad Request - Invalid input parameters
- `404`: Not Found - Resource not found
- `500`: Internal Server Error - Server-side error
- `503`: Service Unavailable - Triton server not ready

---

## Rate Limiting

Currently no rate limiting. In production:
- 1000 requests/minute per IP
- 10000 requests/hour per API key

---

## Versioning

API version is included in the URL: `/api/v1/`

Model versions can be specified in requests. If omitted, latest version is used.

---

## Client Examples

### Python
```python
import requests
import numpy as np

# Price prediction
data = {
    'symbol': '005930',
    'ohlcv_data': np.random.rand(60, 5).tolist()
}
response = requests.post(
    'http://localhost:8005/api/v1/predict/price',
    json=data
)
print(response.json())
```

### cURL
```bash
curl -X POST http://localhost:8005/api/v1/predict/sentiment \
  -H "Content-Type: application/json" \
  -d '{"text": "Samsung reports strong earnings"}'
```

### JavaScript
```javascript
const response = await fetch('http://localhost:8005/api/v1/predict/price', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    symbol: '005930',
    ohlcv_data: [...]  // 60x5 array
  })
});
const data = await response.json();
console.log(data);
```

---

## Best Practices

1. **Batch Requests**: Use batch endpoint for multiple predictions
2. **Error Handling**: Always handle 400/500 errors
3. **Timeouts**: Set reasonable timeouts (2-5 seconds)
4. **Retries**: Implement exponential backoff for retries
5. **Caching**: Cache predictions when appropriate
6. **Monitoring**: Monitor response times and error rates
