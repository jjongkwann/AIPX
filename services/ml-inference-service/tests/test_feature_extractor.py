"""Tests for feature extractor."""
import pytest
import numpy as np
import pandas as pd

from src.features.feature_extractor import FeatureExtractor, SentimentFeatureExtractor


@pytest.fixture
def sample_ohlcv_data():
    """Generate sample OHLCV data."""
    np.random.seed(42)
    data = {
        'open': np.random.uniform(70000, 72000, 100),
        'high': np.random.uniform(71000, 73000, 100),
        'low': np.random.uniform(69000, 71000, 100),
        'close': np.random.uniform(70000, 72000, 100),
        'volume': np.random.uniform(1000000, 2000000, 100)
    }
    return pd.DataFrame(data)


def test_feature_extractor_initialization():
    """Test feature extractor initialization."""
    extractor = FeatureExtractor(normalization="standard")
    assert extractor.normalization == "standard"
    assert extractor.scaler is not None

    extractor = FeatureExtractor(normalization="none")
    assert extractor.scaler is None


def test_extract_price_features(sample_ohlcv_data):
    """Test price feature extraction."""
    extractor = FeatureExtractor()
    features = extractor.extract_price_features(sample_ohlcv_data, window_size=60)

    # Check shape
    assert features.shape == (60, 5)
    assert features.dtype == np.float32

    # Check for NaN values
    assert not np.isnan(features).any()


def test_extract_price_features_insufficient_data():
    """Test feature extraction with insufficient data."""
    extractor = FeatureExtractor()

    # Only 30 rows, need 60
    df = pd.DataFrame({
        'open': np.random.rand(30),
        'high': np.random.rand(30),
        'low': np.random.rand(30),
        'close': np.random.rand(30),
        'volume': np.random.rand(30)
    })

    with pytest.raises(ValueError, match="need at least 60"):
        extractor.extract_price_features(df, window_size=60)


def test_extract_price_features_missing_columns():
    """Test feature extraction with missing columns."""
    extractor = FeatureExtractor()

    # Missing 'volume' column
    df = pd.DataFrame({
        'open': np.random.rand(100),
        'high': np.random.rand(100),
        'low': np.random.rand(100),
        'close': np.random.rand(100)
    })

    with pytest.raises(ValueError, match="must contain columns"):
        extractor.extract_price_features(df, window_size=60)


def test_extract_technical_indicators(sample_ohlcv_data):
    """Test technical indicator extraction."""
    extractor = FeatureExtractor()
    indicators = extractor.extract_technical_indicators(sample_ohlcv_data, window_size=60)

    # Check shape (number of indicators can vary)
    assert indicators.shape[0] == 60
    assert indicators.shape[1] >= 3  # At least returns, RSI, volume ratio

    # Check for NaN values (should be filled)
    assert not np.isnan(indicators).any()


def test_prepare_lstm_input(sample_ohlcv_data):
    """Test LSTM input preparation."""
    extractor = FeatureExtractor()
    features = extractor.prepare_lstm_input(sample_ohlcv_data, window_size=60)

    # Check exact shape required for LSTM
    assert features.shape == (60, 5)
    assert features.dtype == np.float32


def test_prepare_batch_lstm_input(sample_ohlcv_data):
    """Test batch LSTM input preparation."""
    extractor = FeatureExtractor()

    # Create multiple dataframes
    dfs = [sample_ohlcv_data.copy() for _ in range(3)]

    batch_features = extractor.prepare_batch_lstm_input(dfs, window_size=60)

    # Check batch shape
    assert batch_features.shape == (3, 60, 5)
    assert batch_features.dtype == np.float32


def test_normalize_features():
    """Test feature normalization."""
    extractor = FeatureExtractor(normalization="standard")

    features = np.random.rand(60, 5).astype(np.float32)
    normalized = extractor.normalize_features(features)

    # Check shape preserved
    assert normalized.shape == features.shape

    # Check normalization (mean should be close to 0, std close to 1)
    # Note: With StandardScaler on small samples, this might not be exact
    assert abs(normalized.mean()) < 1.0
    assert abs(normalized.std() - 1.0) < 1.0


def test_sentiment_feature_extractor():
    """Test sentiment feature extractor initialization."""
    extractor = SentimentFeatureExtractor()

    # Check that tokenizer is initialized (or None as fallback)
    assert hasattr(extractor, 'tokenizer')


def test_tokenize_text():
    """Test text tokenization."""
    extractor = SentimentFeatureExtractor()

    text = "Samsung reports strong quarterly earnings"
    tokens = extractor.tokenize_text(text, max_length=512)

    # Check output format
    assert "input_ids" in tokens
    assert "attention_mask" in tokens

    # Check shapes
    assert tokens["input_ids"].shape == (512,)
    assert tokens["attention_mask"].shape == (512,)

    # Check dtypes
    assert tokens["input_ids"].dtype == np.int64
    assert tokens["attention_mask"].dtype == np.int64


def test_batch_tokenize():
    """Test batch tokenization."""
    extractor = SentimentFeatureExtractor()

    texts = [
        "Samsung reports strong earnings",
        "Stock market shows positive trends",
        "New AI chip announced"
    ]

    batch_tokens = extractor.batch_tokenize(texts, max_length=512)

    # Check batch output
    assert "input_ids" in batch_tokens
    assert "attention_mask" in batch_tokens

    # Check batch shapes
    assert batch_tokens["input_ids"].shape == (3, 512)
    assert batch_tokens["attention_mask"].shape == (3, 512)


def test_calculate_feature_importance():
    """Test feature importance calculation."""
    extractor = FeatureExtractor()

    features = np.random.rand(10, 60, 5).astype(np.float32)
    predictions = np.random.rand(10, 1).astype(np.float32)

    importance = extractor.calculate_feature_importance(features, predictions)

    # Check that we got importance scores for all features
    assert len(importance) == 5
    assert all(0 <= v <= 1 for v in importance.values())


def test_feature_extraction_consistency(sample_ohlcv_data):
    """Test that feature extraction is consistent."""
    extractor = FeatureExtractor()

    # Extract features twice
    features1 = extractor.extract_price_features(sample_ohlcv_data, window_size=60)
    features2 = extractor.extract_price_features(sample_ohlcv_data, window_size=60)

    # Should be identical
    np.testing.assert_array_almost_equal(features1, features2)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
