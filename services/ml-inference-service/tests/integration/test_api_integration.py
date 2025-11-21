"""Integration tests for FastAPI endpoints."""
import pytest
from httpx import AsyncClient
import numpy as np


@pytest.mark.integration
@pytest.mark.asyncio
class TestPriceAPIIntegration:
    """Test price prediction API endpoints."""
    
    async def test_predict_price_endpoint(self, test_client, sample_market_data):
        """Test POST /api/v1/predict/price endpoint."""
        # Prepare request data
        ohlcv_data = sample_market_data.tail(60)[['open', 'high', 'low', 'close', 'volume']].values.tolist()
        
        request_data = {
            "symbol": "005930",
            "ohlcv_data": ohlcv_data
        }
        
        response = await test_client.post("/api/v1/predict/price", json=request_data)
        
        assert response.status_code == 200
        data = response.json()
        
        assert "predicted_price" in data
        assert "confidence" in data
        assert "model_name" in data
        assert "model_version" in data
        assert "timestamp" in data
        
        assert isinstance(data["predicted_price"], float)
        assert 0 <= data["confidence"] <= 1
    
    async def test_predict_price_with_model_version(self, test_client, sample_market_data):
        """Test price prediction with specific model version."""
        ohlcv_data = sample_market_data.tail(60)[['open', 'high', 'low', 'close', 'volume']].values.tolist()
        
        request_data = {
            "symbol": "005930",
            "ohlcv_data": ohlcv_data,
            "model_version": "1"
        }
        
        response = await test_client.post("/api/v1/predict/price", json=request_data)
        
        assert response.status_code == 200
        data = response.json()
        assert data["model_version"] == "1"
    
    async def test_predict_price_invalid_shape(self, test_client):
        """Test with invalid OHLCV data shape."""
        request_data = {
            "symbol": "005930",
            "ohlcv_data": [[1.0, 1.1, 0.9, 1.05, 1000000.0]] * 30  # Only 30 rows
        }
        
        response = await test_client.post("/api/v1/predict/price", json=request_data)
        
        assert response.status_code == 400
        assert "Expected ohlcv_data shape" in response.json()["detail"]
    
    async def test_predict_price_missing_fields(self, test_client):
        """Test with missing required fields."""
        request_data = {"symbol": "005930"}
        
        response = await test_client.post("/api/v1/predict/price", json=request_data)
        
        assert response.status_code == 422  # Validation error
    
    async def test_predict_price_invalid_data_types(self, test_client):
        """Test with invalid data types."""
        request_data = {
            "symbol": "005930",
            "ohlcv_data": "invalid"  # Should be list
        }
        
        response = await test_client.post("/api/v1/predict/price", json=request_data)
        
        assert response.status_code == 422


@pytest.mark.integration
@pytest.mark.asyncio
class TestSentimentAPIIntegration:
    """Test sentiment analysis API endpoints."""
    
    async def test_predict_sentiment_endpoint(self, test_client):
        """Test POST /api/v1/predict/sentiment endpoint."""
        request_data = {
            "text": "Samsung reports strong quarterly earnings with record-breaking chip sales."
        }
        
        response = await test_client.post("/api/v1/predict/sentiment", json=request_data)
        
        assert response.status_code == 200
        data = response.json()
        
        assert "sentiment" in data
        assert "confidence" in data
        assert "model_name" in data
        assert "model_version" in data
        assert "timestamp" in data
        
        assert -1 <= data["sentiment"] <= 1
        assert 0 <= data["confidence"] <= 1
    
    async def test_predict_sentiment_positive(self, test_client):
        """Test sentiment prediction with positive text."""
        request_data = {
            "text": "Excellent performance! Stock prices soar to new heights. Great news!"
        }
        
        response = await test_client.post("/api/v1/predict/sentiment", json=request_data)
        
        assert response.status_code == 200
        data = response.json()
        
        # Should be positive (or at least not strongly negative)
        assert data["sentiment"] > -0.5
    
    async def test_predict_sentiment_negative(self, test_client):
        """Test sentiment prediction with negative text."""
        request_data = {
            "text": "Company faces severe losses. Stock plummets. Disaster unfolds."
        }
        
        response = await test_client.post("/api/v1/predict/sentiment", json=request_data)
        
        assert response.status_code == 200
        data = response.json()
        
        # Mock will return random sentiment, so just check structure
        assert "sentiment" in data
    
    async def test_predict_sentiment_empty_text(self, test_client):
        """Test with empty text."""
        request_data = {"text": ""}
        
        response = await test_client.post("/api/v1/predict/sentiment", json=request_data)
        
        assert response.status_code == 200
    
    async def test_predict_sentiment_long_text(self, test_client):
        """Test with very long text."""
        request_data = {
            "text": "Samsung Electronics announces new product. " * 200
        }
        
        response = await test_client.post("/api/v1/predict/sentiment", json=request_data)
        
        assert response.status_code == 200


@pytest.mark.integration
@pytest.mark.asyncio
class TestEnsembleAPIIntegration:
    """Test ensemble prediction API endpoints."""
    
    async def test_ensemble_predict_endpoint(self, test_client, sample_market_data):
        """Test POST /api/v1/predict/ensemble endpoint."""
        ohlcv_data = sample_market_data.tail(60)[['open', 'high', 'low', 'close', 'volume']].values.tolist()
        
        request_data = {
            "symbol": "005930",
            "ohlcv_data": ohlcv_data,
            "news_text": "Samsung announces breakthrough in AI chip technology"
        }
        
        response = await test_client.post("/api/v1/predict/ensemble", json=request_data)
        
        assert response.status_code == 200
        data = response.json()
        
        assert "final_prediction" in data
        assert "price_component" in data
        assert "sentiment_component" in data
        assert "confidence" in data
        assert "model_name" in data
        
        assert isinstance(data["final_prediction"], float)
        assert isinstance(data["price_component"], float)
        assert -1 <= data["sentiment_component"] <= 1
        assert 0 <= data["confidence"] <= 1
    
    async def test_ensemble_predict_with_version(self, test_client, sample_market_data):
        """Test ensemble prediction with model version."""
        ohlcv_data = sample_market_data.tail(60)[['open', 'high', 'low', 'close', 'volume']].values.tolist()
        
        request_data = {
            "symbol": "005930",
            "ohlcv_data": ohlcv_data,
            "news_text": "Test news",
            "model_version": "1"
        }
        
        response = await test_client.post("/api/v1/predict/ensemble", json=request_data)
        
        assert response.status_code == 200
        data = response.json()
        assert data["model_version"] == "1"


@pytest.mark.integration
@pytest.mark.asyncio
class TestBatchAPIIntegration:
    """Test batch prediction API endpoints."""
    
    async def test_batch_predict_endpoint(self, test_client, sample_batch_market_data):
        """Test POST /api/v1/predict/batch endpoint."""
        items = []
        for i, df in enumerate(sample_batch_market_data[:3]):
            ohlcv_data = df.tail(60)[['open', 'high', 'low', 'close', 'volume']].values.tolist()
            items.append({
                "symbol": f"00593{i}",
                "ohlcv_data": ohlcv_data
            })
        
        request_data = {"items": items}
        
        response = await test_client.post("/api/v1/predict/batch", json=request_data)
        
        assert response.status_code == 200
        data = response.json()
        
        assert "predictions" in data
        assert "count" in data
        assert "timestamp" in data
        
        assert data["count"] == 3
        assert len(data["predictions"]) == 3
        
        for pred in data["predictions"]:
            assert "predicted_price" in pred
            assert "symbol" in pred
    
    async def test_batch_predict_empty(self, test_client):
        """Test batch prediction with empty list."""
        request_data = {"items": []}
        
        response = await test_client.post("/api/v1/predict/batch", json=request_data)
        
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 0


@pytest.mark.integration
@pytest.mark.asyncio
class TestMetadataAPIIntegration:
    """Test model metadata API endpoints."""
    
    async def test_get_model_metadata(self, test_client):
        """Test GET /api/v1/models/{model_name}/metadata endpoint."""
        response = await test_client.get("/api/v1/models/lstm_price_predictor/metadata")
        
        assert response.status_code == 200
        data = response.json()
        
        assert "name" in data
        assert "versions" in data
        assert "platform" in data
        assert "inputs" in data
        assert "outputs" in data
        assert "max_batch_size" in data
    
    async def test_get_nonexistent_model_metadata(self, test_client, mock_triton_client):
        """Test metadata for non-existent model."""
        # Make mock raise exception
        async def mock_error(model_name):
            raise Exception("Model not found")
        
        mock_triton_client.get_model_metadata = mock_error
        
        response = await test_client.get("/api/v1/models/nonexistent/metadata")
        
        assert response.status_code == 404


@pytest.mark.integration
@pytest.mark.asyncio
class TestMetricsAPIIntegration:
    """Test metrics API endpoints."""
    
    async def test_get_metrics(self, test_client):
        """Test GET /api/v1/metrics endpoint."""
        response = await test_client.get("/api/v1/metrics")
        
        assert response.status_code == 200
        data = response.json()
        
        assert "server_metrics" in data
        assert "app_metrics" in data
        assert "timestamp" in data


@pytest.mark.integration
@pytest.mark.asyncio
class TestHealthCheckIntegration:
    """Test health check endpoints."""
    
    async def test_health_endpoint(self, test_client):
        """Test GET /health endpoint."""
        response = await test_client.get("/health")
        
        assert response.status_code == 200
        data = response.json()
        
        assert "status" in data
        assert "triton_connected" in data
        assert "available_models" in data
        assert "timestamp" in data
        
        assert data["status"] in ["healthy", "degraded", "unhealthy"]
    
    async def test_health_triton_connected(self, test_client):
        """Test health check when Triton is connected."""
        response = await test_client.get("/health")
        
        assert response.status_code == 200
        data = response.json()
        
        assert data["triton_connected"] is True
        assert len(data["available_models"]) > 0


@pytest.mark.integration
@pytest.mark.asyncio
@pytest.mark.slow
class TestConcurrentRequests:
    """Test concurrent request handling."""
    
    async def test_concurrent_price_predictions(self, test_client, sample_market_data):
        """Test multiple concurrent price predictions."""
        import asyncio
        
        ohlcv_data = sample_market_data.tail(60)[['open', 'high', 'low', 'close', 'volume']].values.tolist()
        
        request_data = {
            "symbol": "005930",
            "ohlcv_data": ohlcv_data
        }
        
        # Make 10 concurrent requests
        tasks = [
            test_client.post("/api/v1/predict/price", json=request_data)
            for _ in range(10)
        ]
        
        responses = await asyncio.gather(*tasks)
        
        # All should succeed
        assert all(r.status_code == 200 for r in responses)
        
        # All should return valid predictions
        for response in responses:
            data = response.json()
            assert "predicted_price" in data
    
    async def test_mixed_concurrent_requests(self, test_client, sample_market_data):
        """Test mixed types of concurrent requests."""
        import asyncio
        
        ohlcv_data = sample_market_data.tail(60)[['open', 'high', 'low', 'close', 'volume']].values.tolist()
        
        # Mix of price and sentiment requests
        tasks = [
            test_client.post("/api/v1/predict/price", json={
                "symbol": "005930",
                "ohlcv_data": ohlcv_data
            }),
            test_client.post("/api/v1/predict/sentiment", json={
                "text": "Samsung reports strong earnings"
            }),
            test_client.get("/health"),
            test_client.get("/api/v1/metrics")
        ] * 3
        
        responses = await asyncio.gather(*tasks)
        
        # All should succeed
        assert all(r.status_code == 200 for r in responses)


@pytest.mark.integration
@pytest.mark.asyncio
class TestErrorHandling:
    """Test error handling in API."""
    
    async def test_internal_server_error(self, test_client, mock_triton_client):
        """Test 500 error handling."""
        # Make mock raise exception
        async def mock_error(features, model_version=None):
            raise Exception("Internal error")
        
        mock_triton_client.predict_price = mock_error
        
        ohlcv_data = [[1.0] * 5] * 60
        request_data = {
            "symbol": "005930",
            "ohlcv_data": ohlcv_data
        }
        
        response = await test_client.post("/api/v1/predict/price", json=request_data)
        
        assert response.status_code == 500
        assert "detail" in response.json()
    
    async def test_validation_error_handling(self, test_client):
        """Test 422 validation error handling."""
        # Send invalid request
        response = await test_client.post("/api/v1/predict/price", json={
            "invalid_field": "value"
        })
        
        assert response.status_code == 422
    
    async def test_not_found_error(self, test_client):
        """Test 404 error handling."""
        response = await test_client.get("/api/v1/nonexistent")
        
        assert response.status_code == 404
