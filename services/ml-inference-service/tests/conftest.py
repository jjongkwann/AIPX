"""Pytest fixtures for ML Inference Service tests."""
import pytest
import numpy as np
import pandas as pd
import torch
from datetime import datetime, timedelta
from unittest.mock import Mock, AsyncMock
from httpx import AsyncClient
import asyncio
from typing import AsyncGenerator

# Import application components
import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src'))

from src.config import Settings


@pytest.fixture(scope="session")
def event_loop():
    """Create event loop for async tests."""
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()


@pytest.fixture
def sample_features():
    """Generate sample OHLCV features for testing."""
    np.random.seed(42)
    # Shape: (60, 5) - 60 timesteps, 5 features (O, H, L, C, V normalized)
    features = np.random.randn(60, 5).astype(np.float32)
    return features


@pytest.fixture
def sample_market_data():
    """Generate sample market data DataFrame."""
    np.random.seed(42)
    n_samples = 100
    
    base_price = 71000
    timestamps = pd.date_range('2024-01-01', periods=n_samples, freq='1min')
    
    # Generate realistic price data with trend
    trend = np.linspace(0, 500, n_samples)
    noise = np.random.randn(n_samples) * 100
    close_prices = base_price + trend + noise
    
    # Generate OHLC from close
    open_prices = close_prices + np.random.randn(n_samples) * 50
    high_prices = np.maximum(open_prices, close_prices) + np.abs(np.random.randn(n_samples)) * 30
    low_prices = np.minimum(open_prices, close_prices) - np.abs(np.random.randn(n_samples)) * 30
    volumes = np.random.randint(100000, 1000000, n_samples)
    
    return pd.DataFrame({
        'timestamp': timestamps,
        'open': open_prices,
        'high': high_prices,
        'low': low_prices,
        'close': close_prices,
        'volume': volumes
    })


@pytest.fixture
def sample_batch_market_data():
    """Generate batch of market data."""
    np.random.seed(42)
    batch_size = 8
    batch = []
    
    for i in range(batch_size):
        np.random.seed(42 + i)
        n_samples = 100
        base_price = 71000 + i * 100
        
        timestamps = pd.date_range('2024-01-01', periods=n_samples, freq='1min')
        close_prices = base_price + np.random.randn(n_samples) * 100
        open_prices = close_prices + np.random.randn(n_samples) * 50
        high_prices = np.maximum(open_prices, close_prices) + np.abs(np.random.randn(n_samples)) * 30
        low_prices = np.minimum(open_prices, close_prices) - np.abs(np.random.randn(n_samples)) * 30
        volumes = np.random.randint(100000, 1000000, n_samples)
        
        batch.append(pd.DataFrame({
            'timestamp': timestamps,
            'open': open_prices,
            'high': high_prices,
            'low': low_prices,
            'close': close_prices,
            'volume': volumes
        }))
    
    return batch


@pytest.fixture
def sample_news_text():
    """Sample news text for sentiment analysis."""
    return [
        "Samsung Electronics reports strong quarterly earnings with record-breaking chip sales.",
        "Market concerns grow as inflation rates continue to rise unexpectedly.",
        "Tech sector shows resilience amid global economic uncertainty.",
        "Company announces major layoffs affecting thousands of workers.",
        "Innovation in AI technology drives stock prices to new heights."
    ]


@pytest.fixture
def sample_tokenized_input():
    """Sample tokenized input for sentiment model."""
    np.random.seed(42)
    input_ids = np.random.randint(0, 30000, 512, dtype=np.int64)
    attention_mask = np.ones(512, dtype=np.int64)
    return {
        "input_ids": input_ids,
        "attention_mask": attention_mask
    }


@pytest.fixture
def mock_triton_client(mocker):
    """Mock Triton client for unit tests."""
    client = mocker.Mock()
    client._connected = True
    
    # Mock async methods
    async def mock_connect():
        client._connected = True
    
    async def mock_disconnect():
        client._connected = False
    
    async def mock_is_ready():
        return True
    
    async def mock_predict_price(features, model_version=None):
        # Return realistic prediction
        last_price = features[-1, 0] if len(features) > 0 else 71000.0
        return {
            "predicted_price": float(last_price * 1.01),
            "model_name": "lstm_price_predictor",
            "model_version": model_version or "1",
            "inference_time_ms": 12.5
        }
    
    async def mock_analyze_sentiment(input_ids, attention_mask, model_version=None):
        # Return random sentiment
        return {
            "sentiment": np.random.uniform(-1, 1),
            "confidence": np.random.uniform(0.7, 0.95),
            "model_name": "transformer_sentiment",
            "model_version": model_version or "1"
        }
    
    async def mock_ensemble_predict(price_features, news_text_ids, attention_mask, model_version=None):
        return {
            "final_prediction": 71500.0,
            "price_component": 71300.0,
            "sentiment_component": 0.75,
            "confidence": 0.85,
            "model_name": "ensemble_predictor",
            "model_version": model_version or "1"
        }
    
    async def mock_batch_predict_prices(features_batch, model_version=None):
        results = []
        for i, features in enumerate(features_batch):
            last_price = features[-1, 0] if len(features) > 0 else 71000.0
            results.append({
                "predicted_price": float(last_price * 1.01),
                "batch_index": i,
                "model_version": model_version or "1"
            })
        return results
    
    async def mock_get_model_metadata(model_name):
        return {
            "name": model_name,
            "versions": ["1"],
            "platform": "pytorch_libtorch",
            "inputs": [
                {"name": "input__0", "datatype": "FP32", "shape": [-1, 60, 5]}
            ],
            "outputs": [
                {"name": "output__0", "datatype": "FP32", "shape": [-1, 1]}
            ],
            "max_batch_size": 32
        }
    
    async def mock_get_server_metrics():
        return {
            "lstm_price_predictor": {
                "count": 100,
                "total_time_ns": 1250000000,
                "avg_time_ms": 12.5,
                "version": "1"
            }
        }
    
    client.connect = mock_connect
    client.disconnect = mock_disconnect
    client.is_ready = mock_is_ready
    client.predict_price = mock_predict_price
    client.analyze_sentiment = mock_analyze_sentiment
    client.ensemble_predict = mock_ensemble_predict
    client.batch_predict_prices = mock_batch_predict_prices
    client.get_model_metadata = mock_get_model_metadata
    client.get_server_metrics = mock_get_server_metrics
    
    return client


@pytest.fixture
async def test_client(mock_triton_client, mocker):
    """Provide test HTTP client for FastAPI."""
    # Patch triton_client in main module
    mocker.patch('src.main.triton_client', mock_triton_client)
    
    from src.main import app
    
    async with AsyncClient(app=app, base_url="http://test") as client:
        yield client


@pytest.fixture
def mock_lstm_model():
    """Mock LSTM model for testing."""
    from src.models.lstm_model import LSTMPredictor
    
    model = LSTMPredictor(
        input_size=5,
        hidden_size=64,
        num_layers=2,
        dropout=0.2
    )
    model.eval()
    return model


@pytest.fixture
def sample_torch_tensor():
    """Sample PyTorch tensor for model testing."""
    torch.manual_seed(42)
    # Shape: (batch=4, seq_len=60, features=5)
    return torch.randn(4, 60, 5)


@pytest.fixture
def test_settings():
    """Test settings configuration."""
    return Settings(
        triton_url="localhost:8001",
        db_password="test_password",
        feature_window_size=60,
        feature_normalization="standard",
        max_batch_size=32,
        log_level="DEBUG"
    )


@pytest.fixture
def mock_feature_extractor(mocker):
    """Mock feature extractor."""
    extractor = mocker.Mock()
    
    def mock_extract_price_features(df, window_size=60):
        np.random.seed(42)
        return np.random.randn(window_size, 5).astype(np.float32)
    
    def mock_prepare_lstm_input(df, window_size=None):
        window_size = window_size or 60
        return np.random.randn(window_size, 5).astype(np.float32)
    
    extractor.extract_price_features = mock_extract_price_features
    extractor.prepare_lstm_input = mock_prepare_lstm_input
    
    return extractor


@pytest.fixture
def mock_sentiment_extractor(mocker):
    """Mock sentiment feature extractor."""
    extractor = mocker.Mock()
    
    def mock_tokenize_text(text, max_length=512):
        return {
            "input_ids": np.zeros(max_length, dtype=np.int64),
            "attention_mask": np.ones(max_length, dtype=np.int64)
        }
    
    extractor.tokenize_text = mock_tokenize_text
    
    return extractor


@pytest.fixture
def sample_prediction_results():
    """Sample prediction results for testing."""
    return {
        "price_prediction": {
            "predicted_price": 71500.0,
            "confidence": 0.85,
            "model_name": "lstm_price_predictor",
            "model_version": "1"
        },
        "sentiment_prediction": {
            "sentiment": 0.75,
            "confidence": 0.90,
            "model_name": "transformer_sentiment",
            "model_version": "1"
        },
        "ensemble_prediction": {
            "final_prediction": 71600.0,
            "price_component": 71500.0,
            "sentiment_component": 0.75,
            "confidence": 0.88,
            "model_name": "ensemble_predictor",
            "model_version": "1"
        }
    }


@pytest.fixture
def performance_metrics():
    """Sample performance metrics."""
    return {
        "latency_ms": {
            "mean": 12.5,
            "median": 11.0,
            "p95": 18.0,
            "p99": 22.0,
            "min": 8.0,
            "max": 35.0
        },
        "throughput": {
            "requests_per_second": 550,
            "successful_requests": 5500,
            "failed_requests": 5,
            "error_rate": 0.0009
        }
    }


# Markers for test categorization
def pytest_configure(config):
    """Configure custom pytest markers."""
    config.addinivalue_line("markers", "unit: Unit tests")
    config.addinivalue_line("markers", "integration: Integration tests")
    config.addinivalue_line("markers", "performance: Performance tests")
    config.addinivalue_line("markers", "validation: Model validation tests")
    config.addinivalue_line("markers", "monitoring: Monitoring tests")
    config.addinivalue_line("markers", "slow: Slow running tests")
    config.addinivalue_line("markers", "gpu: Tests requiring GPU")


# Async helper functions
@pytest.fixture
async def async_timeout():
    """Helper for async test timeouts."""
    async def _timeout(coro, timeout_sec=5):
        try:
            return await asyncio.wait_for(coro, timeout=timeout_sec)
        except asyncio.TimeoutError:
            pytest.fail(f"Test timed out after {timeout_sec} seconds")
    return _timeout
