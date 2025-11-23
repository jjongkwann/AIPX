"""Unit tests for Triton client."""

from unittest.mock import AsyncMock, Mock

import numpy as np
import pytest
from tritonclient.utils import InferenceServerException

from src.triton_client import TritonClient


@pytest.mark.unit
@pytest.mark.asyncio
class TestTritonClient:
    """Test TritonClient class."""

    async def test_client_initialization(self):
        """Test client initialization."""
        client = TritonClient(url="localhost:8001")
        assert client.url == "localhost:8001"
        assert client.client is None
        assert not client._connected

    async def test_connect_success(self, mocker):
        """Test successful connection to Triton server."""
        # Mock grpc client
        mock_client = AsyncMock()
        mock_client.is_server_live.return_value = True
        mock_client.is_server_ready.return_value = True

        # Mock model repository
        mock_model = Mock()
        mock_model.name = "test_model"
        mock_client.get_model_repository_index.return_value = [mock_model]

        mocker.patch("src.triton_client.aiogrpcclient.InferenceServerClient", return_value=mock_client)

        client = TritonClient()
        await client.connect()

        assert client._connected
        assert client.client is not None

    async def test_connect_failure(self, mocker):
        """Test connection failure handling."""
        mock_client = AsyncMock()
        mock_client.is_server_live.side_effect = Exception("Connection failed")

        mocker.patch("src.triton_client.aiogrpcclient.InferenceServerClient", return_value=mock_client)

        client = TritonClient()

        with pytest.raises(Exception, match="Connection failed"):
            await client.connect()

        assert not client._connected

    async def test_disconnect(self, mocker):
        """Test disconnection."""
        mock_client = AsyncMock()
        mock_client.close = AsyncMock()

        client = TritonClient()
        client.client = mock_client
        client._connected = True

        await client.disconnect()

        assert not client._connected
        mock_client.close.assert_called_once()

    async def test_is_ready_connected(self, mocker):
        """Test is_ready when connected."""
        mock_client = AsyncMock()
        mock_client.is_server_ready.return_value = True

        client = TritonClient()
        client.client = mock_client

        result = await client.is_ready()
        assert result is True

    async def test_is_ready_not_connected(self):
        """Test is_ready when not connected."""
        client = TritonClient()
        result = await client.is_ready()
        assert result is False

    async def test_predict_price_valid_input(self, mocker):
        """Test price prediction with valid input."""
        # Create mock client
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.as_numpy.return_value = np.array([71500.0])
        mock_client.infer.return_value = mock_response

        # Mock statistics
        mock_stats = Mock()
        mock_model_stat = Mock()
        mock_inference_stat = Mock()
        mock_success_stat = Mock()
        mock_success_stat.ns = 12500000  # 12.5ms
        mock_inference_stat.success = mock_success_stat
        mock_model_stat.inference_stats = mock_inference_stat
        mock_stats.model_stats = [mock_model_stat]
        mock_client.get_inference_statistics.return_value = mock_stats

        client = TritonClient()
        client.client = mock_client
        client._connected = True

        # Test input
        features = np.random.randn(60, 5).astype(np.float32)

        result = await client.predict_price(features)

        assert "predicted_price" in result
        assert result["predicted_price"] == 71500.0
        assert result["model_name"] == "lstm_price_predictor"
        assert result["model_version"] == "latest"

    async def test_predict_price_invalid_shape(self):
        """Test price prediction with invalid shape."""
        client = TritonClient()
        client._connected = True

        # Invalid shape
        features = np.random.randn(50, 5).astype(np.float32)

        with pytest.raises(ValueError, match="Expected features shape"):
            await client.predict_price(features)

    async def test_predict_price_not_connected(self, mocker):
        """Test price prediction auto-connects."""
        mock_client = AsyncMock()
        mock_client.is_server_live.return_value = True
        mock_client.is_server_ready.return_value = True
        mock_client.get_model_repository_index.return_value = []

        mock_response = Mock()
        mock_response.as_numpy.return_value = np.array([71500.0])
        mock_client.infer.return_value = mock_response

        mock_stats = Mock()
        mock_stats.model_stats = []
        mock_client.get_inference_statistics.return_value = mock_stats

        mocker.patch("src.triton_client.aiogrpcclient.InferenceServerClient", return_value=mock_client)

        client = TritonClient()
        features = np.random.randn(60, 5).astype(np.float32)

        result = await client.predict_price(features)

        assert client._connected
        assert "predicted_price" in result

    async def test_analyze_sentiment_valid_input(self, mocker):
        """Test sentiment analysis with valid input."""
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.as_numpy.side_effect = [
            np.array([0.75]),  # sentiment
            np.array([0.90]),  # confidence
        ]
        mock_client.infer.return_value = mock_response

        client = TritonClient()
        client.client = mock_client
        client._connected = True

        input_ids = np.zeros(512, dtype=np.int64)
        attention_mask = np.ones(512, dtype=np.int64)

        result = await client.analyze_sentiment(input_ids, attention_mask)

        assert result["sentiment"] == 0.75
        assert result["confidence"] == 0.90
        assert result["model_name"] == "transformer_sentiment"

    async def test_analyze_sentiment_invalid_shape(self):
        """Test sentiment analysis with invalid shape."""
        client = TritonClient()
        client._connected = True

        # Invalid shapes
        input_ids = np.zeros(256, dtype=np.int64)
        attention_mask = np.ones(512, dtype=np.int64)

        with pytest.raises(ValueError, match="Expected input_ids shape"):
            await client.analyze_sentiment(input_ids, attention_mask)

    async def test_ensemble_predict(self, mocker):
        """Test ensemble prediction."""
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.as_numpy.side_effect = [
            np.array([71600.0]),  # final_prediction
            np.array([71500.0]),  # price_component
            np.array([0.75]),  # sentiment_component
            np.array([0.88]),  # confidence
        ]
        mock_client.infer.return_value = mock_response

        client = TritonClient()
        client.client = mock_client
        client._connected = True

        price_features = np.random.randn(60, 5).astype(np.float32)
        news_text_ids = np.zeros(512, dtype=np.int64)
        attention_mask = np.ones(512, dtype=np.int64)

        result = await client.ensemble_predict(price_features, news_text_ids, attention_mask)

        assert result["final_prediction"] == 71600.0
        assert result["price_component"] == 71500.0
        assert result["sentiment_component"] == 0.75
        assert result["confidence"] == 0.88

    async def test_batch_predict_prices(self, mocker):
        """Test batch price prediction."""
        mock_client = AsyncMock()
        mock_response = Mock()
        # Return 3 predictions
        mock_response.as_numpy.return_value = np.array([[71500.0], [71600.0], [71700.0]])
        mock_client.infer.return_value = mock_response

        client = TritonClient()
        client.client = mock_client
        client._connected = True

        features_batch = [
            np.random.randn(60, 5).astype(np.float32),
            np.random.randn(60, 5).astype(np.float32),
            np.random.randn(60, 5).astype(np.float32),
        ]

        results = await client.batch_predict_prices(features_batch)

        assert len(results) == 3
        assert results[0]["predicted_price"] == 71500.0
        assert results[1]["predicted_price"] == 71600.0
        assert results[2]["predicted_price"] == 71700.0
        assert all(r["batch_index"] == i for i, r in enumerate(results))

    async def test_get_model_metadata(self, mocker):
        """Test getting model metadata."""
        mock_client = AsyncMock()

        # Mock metadata
        mock_metadata = Mock()
        mock_metadata.name = "lstm_price_predictor"
        mock_metadata.versions = ["1", "2"]
        mock_metadata.platform = "pytorch_libtorch"

        mock_input = Mock()
        mock_input.name = "input__0"
        mock_input.datatype = "FP32"
        mock_input.shape = [-1, 60, 5]
        mock_metadata.inputs = [mock_input]

        mock_output = Mock()
        mock_output.name = "output__0"
        mock_output.datatype = "FP32"
        mock_output.shape = [-1, 1]
        mock_metadata.outputs = [mock_output]

        # Mock config
        mock_config = Mock()
        mock_config.config.max_batch_size = 32

        mock_client.get_model_metadata.return_value = mock_metadata
        mock_client.get_model_config.return_value = mock_config

        client = TritonClient()
        client.client = mock_client
        client._connected = True

        metadata = await client.get_model_metadata("lstm_price_predictor")

        assert metadata["name"] == "lstm_price_predictor"
        assert metadata["versions"] == ["1", "2"]
        assert metadata["platform"] == "pytorch_libtorch"
        assert metadata["max_batch_size"] == 32
        assert len(metadata["inputs"]) == 1
        assert len(metadata["outputs"]) == 1

    async def test_get_server_metrics(self, mocker):
        """Test getting server metrics."""
        mock_client = AsyncMock()

        # Mock statistics
        mock_stats = Mock()
        mock_model_stat = Mock()
        mock_model_stat.name = "lstm_price_predictor"
        mock_model_stat.version = "1"

        mock_success = Mock()
        mock_success.count = 100
        mock_success.ns = 1250000000  # 1.25 seconds total

        mock_inference_stats = Mock()
        mock_inference_stats.success = mock_success
        mock_model_stat.inference_stats = mock_inference_stats

        mock_stats.model_stats = [mock_model_stat]
        mock_client.get_inference_statistics.return_value = mock_stats

        client = TritonClient()
        client.client = mock_client
        client._connected = True

        metrics = await client.get_server_metrics()

        assert "lstm_price_predictor" in metrics
        assert metrics["lstm_price_predictor"]["count"] == 100
        assert metrics["lstm_price_predictor"]["avg_time_ms"] == 12.5

    async def test_inference_server_exception(self, mocker):
        """Test handling of InferenceServerException."""
        mock_client = AsyncMock()
        mock_client.infer.side_effect = InferenceServerException("Model not found")

        client = TritonClient()
        client.client = mock_client
        client._connected = True

        features = np.random.randn(60, 5).astype(np.float32)

        with pytest.raises(InferenceServerException):
            await client.predict_price(features)

    async def test_timeout_handling(self, mocker):
        """Test timeout handling."""
        mock_client = AsyncMock()
        mock_client.infer.side_effect = TimeoutError("Request timeout")

        client = TritonClient()
        client.client = mock_client
        client._connected = True

        features = np.random.randn(60, 5).astype(np.float32)

        with pytest.raises(Exception):
            await client.predict_price(features)
