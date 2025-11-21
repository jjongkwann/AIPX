"""Unit tests for API schemas."""
import pytest
from pydantic import ValidationError
from datetime import datetime

from src.main import (
    PredictPriceRequest,
    PredictPriceResponse,
    SentimentRequest,
    SentimentResponse,
    EnsemblePredictRequest,
    EnsemblePredictResponse,
    BatchPredictRequest,
    HealthResponse
)


@pytest.mark.unit
class TestPredictPriceRequest:
    """Test PredictPriceRequest schema."""
    
    def test_valid_request(self):
        """Test valid price prediction request."""
        data = {
            "symbol": "005930",
            "ohlcv_data": [[1.0, 1.1, 0.9, 1.05, 1000000.0]] * 60
        }
        
        request = PredictPriceRequest(**data)
        assert request.symbol == "005930"
        assert len(request.ohlcv_data) == 60
    
    def test_with_model_version(self):
        """Test request with model version."""
        data = {
            "symbol": "005930",
            "ohlcv_data": [[1.0, 1.1, 0.9, 1.05, 1000000.0]] * 60,
            "model_version": "2"
        }
        
        request = PredictPriceRequest(**data)
        assert request.model_version == "2"
    
    def test_missing_symbol(self):
        """Test missing symbol field."""
        data = {
            "ohlcv_data": [[1.0, 1.1, 0.9, 1.05, 1000000.0]] * 60
        }
        
        with pytest.raises(ValidationError):
            PredictPriceRequest(**data)
    
    def test_missing_ohlcv_data(self):
        """Test missing OHLCV data."""
        data = {"symbol": "005930"}
        
        with pytest.raises(ValidationError):
            PredictPriceRequest(**data)
    
    def test_invalid_ohlcv_shape(self):
        """Test invalid OHLCV data shape."""
        data = {
            "symbol": "005930",
            "ohlcv_data": [[1.0, 1.1]] * 60  # Only 2 columns
        }
        
        # Schema allows this, but application should validate
        request = PredictPriceRequest(**data)
        assert len(request.ohlcv_data[0]) == 2


@pytest.mark.unit
class TestPredictPriceResponse:
    """Test PredictPriceResponse schema."""
    
    def test_valid_response(self):
        """Test valid price prediction response."""
        data = {
            "predicted_price": 71500.0,
            "confidence": 0.85,
            "model_name": "lstm_price_predictor",
            "model_version": "1",
            "inference_time_ms": 12.5,
            "timestamp": datetime.utcnow()
        }
        
        response = PredictPriceResponse(**data)
        assert response.predicted_price == 71500.0
        assert response.confidence == 0.85
    
    def test_without_inference_time(self):
        """Test response without inference time."""
        data = {
            "predicted_price": 71500.0,
            "confidence": 0.85,
            "model_name": "lstm_price_predictor",
            "model_version": "1",
            "timestamp": datetime.utcnow()
        }
        
        response = PredictPriceResponse(**data)
        assert response.inference_time_ms is None


@pytest.mark.unit
class TestSentimentRequest:
    """Test SentimentRequest schema."""
    
    def test_valid_request(self):
        """Test valid sentiment request."""
        data = {
            "text": "Samsung reports strong earnings"
        }
        
        request = SentimentRequest(**data)
        assert request.text == "Samsung reports strong earnings"
    
    def test_empty_text(self):
        """Test with empty text."""
        data = {"text": ""}
        
        request = SentimentRequest(**data)
        assert request.text == ""
    
    def test_long_text(self):
        """Test with very long text."""
        data = {"text": "A" * 10000}
        
        request = SentimentRequest(**data)
        assert len(request.text) == 10000
    
    def test_missing_text(self):
        """Test missing text field."""
        with pytest.raises(ValidationError):
            SentimentRequest()


@pytest.mark.unit
class TestSentimentResponse:
    """Test SentimentResponse schema."""
    
    def test_valid_response(self):
        """Test valid sentiment response."""
        data = {
            "sentiment": 0.75,
            "confidence": 0.90,
            "model_name": "transformer_sentiment",
            "model_version": "1",
            "timestamp": datetime.utcnow()
        }
        
        response = SentimentResponse(**data)
        assert response.sentiment == 0.75
        assert response.confidence == 0.90
    
    def test_negative_sentiment(self):
        """Test negative sentiment value."""
        data = {
            "sentiment": -0.5,
            "confidence": 0.80,
            "model_name": "transformer_sentiment",
            "model_version": "1",
            "timestamp": datetime.utcnow()
        }
        
        response = SentimentResponse(**data)
        assert response.sentiment == -0.5


@pytest.mark.unit
class TestEnsemblePredictRequest:
    """Test EnsemblePredictRequest schema."""
    
    def test_valid_request(self):
        """Test valid ensemble request."""
        data = {
            "symbol": "005930",
            "ohlcv_data": [[1.0, 1.1, 0.9, 1.05, 1000000.0]] * 60,
            "news_text": "Samsung announces new chip"
        }
        
        request = EnsemblePredictRequest(**data)
        assert request.symbol == "005930"
        assert request.news_text == "Samsung announces new chip"
    
    def test_with_model_version(self):
        """Test with model version."""
        data = {
            "symbol": "005930",
            "ohlcv_data": [[1.0, 1.1, 0.9, 1.05, 1000000.0]] * 60,
            "news_text": "Test news",
            "model_version": "2"
        }
        
        request = EnsemblePredictRequest(**data)
        assert request.model_version == "2"


@pytest.mark.unit
class TestEnsemblePredictResponse:
    """Test EnsemblePredictResponse schema."""
    
    def test_valid_response(self):
        """Test valid ensemble response."""
        data = {
            "final_prediction": 71600.0,
            "price_component": 71500.0,
            "sentiment_component": 0.75,
            "confidence": 0.88,
            "model_name": "ensemble_predictor",
            "model_version": "1",
            "timestamp": datetime.utcnow()
        }
        
        response = EnsemblePredictResponse(**data)
        assert response.final_prediction == 71600.0
        assert response.price_component == 71500.0
        assert response.sentiment_component == 0.75


@pytest.mark.unit
class TestBatchPredictRequest:
    """Test BatchPredictRequest schema."""
    
    def test_valid_request(self):
        """Test valid batch request."""
        data = {
            "items": [
                {"symbol": "005930", "ohlcv_data": [[1.0] * 5] * 60},
                {"symbol": "000660", "ohlcv_data": [[1.0] * 5] * 60}
            ]
        }
        
        request = BatchPredictRequest(**data)
        assert len(request.items) == 2
    
    def test_empty_batch(self):
        """Test empty batch."""
        data = {"items": []}
        
        request = BatchPredictRequest(**data)
        assert len(request.items) == 0


@pytest.mark.unit
class TestHealthResponse:
    """Test HealthResponse schema."""
    
    def test_healthy_status(self):
        """Test healthy status response."""
        data = {
            "status": "healthy",
            "triton_connected": True,
            "available_models": ["lstm_price_predictor", "transformer_sentiment"],
            "timestamp": datetime.utcnow()
        }
        
        response = HealthResponse(**data)
        assert response.status == "healthy"
        assert response.triton_connected
        assert len(response.available_models) == 2
    
    def test_unhealthy_status(self):
        """Test unhealthy status response."""
        data = {
            "status": "unhealthy",
            "triton_connected": False,
            "available_models": [],
            "timestamp": datetime.utcnow()
        }
        
        response = HealthResponse(**data)
        assert response.status == "unhealthy"
        assert not response.triton_connected
