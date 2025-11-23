"""Tests for Triton client."""

from unittest.mock import AsyncMock, Mock, patch

import numpy as np
import pytest

from src.triton_client import TritonClient


@pytest.fixture
def mock_triton_client():
    """Mock Triton gRPC client."""
    with patch("tritonclient.grpc.aio.InferenceServerClient") as mock:
        client = Mock()
        client.is_server_live = AsyncMock(return_value=True)
        client.is_server_ready = AsyncMock(return_value=True)
        client.get_model_repository_index = AsyncMock(
            return_value=[Mock(name="lstm_price_predictor"), Mock(name="transformer_sentiment")]
        )
        mock.return_value = client
        yield mock


@pytest.mark.asyncio
async def test_connect(mock_triton_client):
    """Test connection to Triton server."""
    client = TritonClient("triton:8001")
    await client.connect()

    assert client._connected is True
    mock_triton_client.assert_called_once()


@pytest.mark.asyncio
async def test_predict_price():
    """Test price prediction."""
    client = TritonClient()
    client._connected = True

    # Mock the client
    mock_response = Mock()
    mock_response.as_numpy = Mock(return_value=np.array([[71500.0]]))

    mock_client = AsyncMock()
    mock_client.infer = AsyncMock(return_value=mock_response)
    mock_client.get_inference_statistics = AsyncMock(return_value=Mock(model_stats=[]))

    client.client = mock_client

    # Test prediction
    features = np.random.rand(60, 5).astype(np.float32)
    result = await client.predict_price(features)

    assert "predicted_price" in result
    assert isinstance(result["predicted_price"], float)
    assert result["model_name"] == "lstm_price_predictor"


@pytest.mark.asyncio
async def test_predict_price_invalid_shape():
    """Test price prediction with invalid input shape."""
    client = TritonClient()
    client._connected = True

    # Wrong shape
    features = np.random.rand(30, 5).astype(np.float32)

    with pytest.raises(ValueError, match="Expected features shape"):
        await client.predict_price(features)


@pytest.mark.asyncio
async def test_analyze_sentiment():
    """Test sentiment analysis."""
    client = TritonClient()
    client._connected = True

    # Mock response
    mock_response = Mock()
    mock_response.as_numpy = Mock(
        side_effect=[
            np.array([0.78]),  # sentiment
            np.array([0.92]),  # confidence
        ]
    )

    mock_client = AsyncMock()
    mock_client.infer = AsyncMock(return_value=mock_response)

    client.client = mock_client

    # Test sentiment analysis
    input_ids = np.zeros(512, dtype=np.int64)
    attention_mask = np.ones(512, dtype=np.int64)

    result = await client.analyze_sentiment(input_ids, attention_mask)

    assert "sentiment" in result
    assert "confidence" in result
    assert result["model_name"] == "transformer_sentiment"


@pytest.mark.asyncio
async def test_ensemble_predict():
    """Test ensemble prediction."""
    client = TritonClient()
    client._connected = True

    # Mock response
    mock_response = Mock()
    mock_response.as_numpy = Mock(
        side_effect=[
            np.array([72000.0]),  # final_prediction
            np.array([71500.0]),  # price_component
            np.array([0.78]),  # sentiment_component
            np.array([0.88]),  # confidence
        ]
    )

    mock_client = AsyncMock()
    mock_client.infer = AsyncMock(return_value=mock_response)

    client.client = mock_client

    # Test ensemble
    price_features = np.random.rand(60, 5).astype(np.float32)
    news_ids = np.zeros(512, dtype=np.int64)
    attention_mask = np.ones(512, dtype=np.int64)

    result = await client.ensemble_predict(price_features, news_ids, attention_mask)

    assert "final_prediction" in result
    assert "price_component" in result
    assert "sentiment_component" in result
    assert "confidence" in result


@pytest.mark.asyncio
async def test_batch_predict_prices():
    """Test batch price prediction."""
    client = TritonClient()
    client._connected = True

    # Mock response
    mock_response = Mock()
    mock_response.as_numpy = Mock(return_value=np.array([[71500.0], [72000.0], [71800.0]]))

    mock_client = AsyncMock()
    mock_client.infer = AsyncMock(return_value=mock_response)

    client.client = mock_client

    # Test batch prediction
    features_batch = [np.random.rand(60, 5).astype(np.float32) for _ in range(3)]

    results = await client.batch_predict_prices(features_batch)

    assert len(results) == 3
    assert all("predicted_price" in r for r in results)


@pytest.mark.asyncio
async def test_is_ready():
    """Test server readiness check."""
    client = TritonClient()

    # Not connected
    assert await client.is_ready() is False

    # Connected
    mock_client = AsyncMock()
    mock_client.is_server_ready = AsyncMock(return_value=True)
    client.client = mock_client

    assert await client.is_ready() is True


@pytest.mark.asyncio
async def test_get_model_metadata():
    """Test getting model metadata."""
    client = TritonClient()
    client._connected = True

    # Mock metadata
    mock_metadata = Mock()
    mock_metadata.name = "lstm_price_predictor"
    mock_metadata.versions = ["1"]
    mock_metadata.platform = "pytorch_libtorch"
    mock_metadata.inputs = [Mock(name="input__0", datatype="FP32", shape=[60, 5])]
    mock_metadata.outputs = [Mock(name="output__0", datatype="FP32", shape=[1])]

    mock_config = Mock()
    mock_config.config.max_batch_size = 32

    mock_client = AsyncMock()
    mock_client.get_model_metadata = AsyncMock(return_value=mock_metadata)
    mock_client.get_model_config = AsyncMock(return_value=mock_config)

    client.client = mock_client

    # Test metadata retrieval
    metadata = await client.get_model_metadata("lstm_price_predictor")

    assert metadata["name"] == "lstm_price_predictor"
    assert metadata["platform"] == "pytorch_libtorch"
    assert len(metadata["inputs"]) == 1
    assert metadata["max_batch_size"] == 32


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
