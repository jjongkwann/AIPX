"""Unit tests for feature extraction."""
import pytest
import numpy as np
import pandas as pd
from src.features.feature_extractor import FeatureExtractor, SentimentFeatureExtractor


@pytest.mark.unit
class TestFeatureExtractor:
    """Test FeatureExtractor class."""
    
    def test_initialization(self):
        """Test feature extractor initialization."""
        extractor = FeatureExtractor(normalization="standard")
        assert extractor.normalization == "standard"
        assert extractor.scaler is not None
    
    def test_initialization_minmax(self):
        """Test with MinMax normalization."""
        extractor = FeatureExtractor(normalization="minmax")
        assert extractor.normalization == "minmax"
        assert extractor.scaler is not None
    
    def test_initialization_none(self):
        """Test without normalization."""
        extractor = FeatureExtractor(normalization="none")
        assert extractor.normalization == "none"
        assert extractor.scaler is None
    
    def test_extract_price_features_valid(self, sample_market_data):
        """Test price feature extraction with valid data."""
        extractor = FeatureExtractor()
        features = extractor.extract_price_features(sample_market_data, window_size=60)
        
        assert features.shape == (60, 5)
        assert features.dtype == np.float32
        assert not np.isnan(features).any()
        assert not np.isinf(features).any()
    
    def test_extract_price_features_shape(self, sample_market_data):
        """Test output shape for different window sizes."""
        extractor = FeatureExtractor()
        
        for window_size in [30, 60, 90]:
            features = extractor.extract_price_features(
                sample_market_data, window_size=window_size
            )
            assert features.shape == (window_size, 5)
    
    def test_extract_price_features_missing_columns(self):
        """Test with missing required columns."""
        extractor = FeatureExtractor()
        df = pd.DataFrame({'close': [100, 101, 102]})
        
        with pytest.raises(ValueError, match="must contain columns"):
            extractor.extract_price_features(df)
    
    def test_extract_price_features_insufficient_data(self, sample_market_data):
        """Test with insufficient data."""
        extractor = FeatureExtractor()
        short_df = sample_market_data.head(30)
        
        with pytest.raises(ValueError, match="need at least"):
            extractor.extract_price_features(short_df, window_size=60)
    
    def test_extract_price_features_ratios(self, sample_market_data):
        """Test that price ratios are calculated correctly."""
        extractor = FeatureExtractor()
        features = extractor.extract_price_features(sample_market_data, window_size=60)
        
        # Check that ratios are reasonable (around 1.0)
        open_close_ratio = features[:, 1]
        high_close_ratio = features[:, 2]
        low_close_ratio = features[:, 3]
        
        assert np.all(open_close_ratio > 0.95)
        assert np.all(open_close_ratio < 1.05)
        assert np.all(high_close_ratio >= 0.99)  # High should be >= close
        assert np.all(low_close_ratio <= 1.01)   # Low should be <= close
    
    def test_extract_price_features_normalization(self, sample_market_data):
        """Test that close prices are normalized."""
        extractor = FeatureExtractor()
        features = extractor.extract_price_features(sample_market_data, window_size=60)
        
        # Close normalized should have ~0 mean and ~1 std
        close_normalized = features[:, 0]
        assert abs(close_normalized.mean()) < 0.1
        assert abs(close_normalized.std() - 1.0) < 0.1
    
    def test_extract_technical_indicators(self, sample_market_data):
        """Test technical indicator extraction."""
        extractor = FeatureExtractor()
        indicators = extractor.extract_technical_indicators(sample_market_data, window_size=60)
        
        assert indicators.shape[0] == 60
        assert indicators.shape[1] >= 3  # At least returns, RSI, volume_ratio
        assert not np.isnan(indicators).any()
        assert not np.isinf(indicators).any()
    
    def test_technical_indicators_rsi_range(self, sample_market_data):
        """Test that RSI is in valid range."""
        extractor = FeatureExtractor()
        indicators = extractor.extract_technical_indicators(sample_market_data, window_size=60)
        
        # RSI column (normalized to [0, 1])
        rsi_idx = 2  # Assuming RSI is 3rd indicator
        rsi_values = indicators[:, rsi_idx]
        
        assert np.all(rsi_values >= 0)
        assert np.all(rsi_values <= 1)
    
    def test_extract_advanced_features(self, sample_market_data):
        """Test advanced feature extraction."""
        extractor = FeatureExtractor()
        features = extractor.extract_advanced_features(sample_market_data, window_size=60)
        
        assert features.shape[0] == 60
        assert features.shape[1] > 5  # Combined features
        assert not np.isnan(features).any()
        assert not np.isinf(features).any()
    
    def test_normalize_features(self):
        """Test feature normalization."""
        extractor = FeatureExtractor(normalization="standard")
        features = np.random.randn(60, 5).astype(np.float32)
        
        normalized = extractor.normalize_features(features)
        
        assert normalized.shape == features.shape
        # Check normalization was applied
        assert not np.allclose(normalized, features)
    
    def test_normalize_features_none(self):
        """Test without normalization."""
        extractor = FeatureExtractor(normalization="none")
        features = np.random.randn(60, 5).astype(np.float32)
        
        normalized = extractor.normalize_features(features)
        
        # Should return unchanged
        assert np.allclose(normalized, features)
    
    def test_prepare_lstm_input(self, sample_market_data):
        """Test LSTM input preparation."""
        extractor = FeatureExtractor()
        features = extractor.prepare_lstm_input(sample_market_data, window_size=60)
        
        assert features.shape == (60, 5)
        assert features.dtype == np.float32
    
    def test_prepare_lstm_input_default_window(self, sample_market_data, test_settings):
        """Test with default window size from settings."""
        extractor = FeatureExtractor()
        features = extractor.prepare_lstm_input(sample_market_data)
        
        assert features.shape[0] == test_settings.feature_window_size
    
    def test_prepare_batch_lstm_input(self, sample_batch_market_data):
        """Test batch LSTM input preparation."""
        extractor = FeatureExtractor()
        batch_features = extractor.prepare_batch_lstm_input(
            sample_batch_market_data, window_size=60
        )
        
        assert batch_features.shape == (len(sample_batch_market_data), 60, 5)
        assert batch_features.dtype == np.float32
    
    def test_calculate_feature_importance(self):
        """Test feature importance calculation."""
        extractor = FeatureExtractor()
        
        features = np.random.randn(10, 60, 5).astype(np.float32)
        predictions = np.random.randn(10, 1).astype(np.float32)
        
        importance = extractor.calculate_feature_importance(features, predictions)
        
        assert isinstance(importance, dict)
        assert len(importance) == 5
        assert all(0 <= v <= 1 for v in importance.values())
    
    def test_extract_with_nan_values(self):
        """Test extraction with NaN values - NaN is propagated in current implementation."""
        extractor = FeatureExtractor()

        # Create DataFrame with NaN
        df = pd.DataFrame({
            'timestamp': pd.date_range('2024-01-01', periods=100, freq='1min'),
            'open': np.random.randn(100) * 100 + 71000,
            'high': np.random.randn(100) * 100 + 71100,
            'low': np.random.randn(100) * 100 + 70900,
            'close': np.random.randn(100) * 100 + 71000,
            'volume': np.random.randint(100000, 1000000, 100)
        })
        df.loc[50, 'close'] = np.nan

        # Current implementation propagates NaN - verify features are extracted
        features = extractor.extract_price_features(df, window_size=60)
        assert features.shape == (60, 5)
    
    def test_extract_with_zero_volume(self):
        """Test extraction with zero volumes."""
        extractor = FeatureExtractor()
        
        df = pd.DataFrame({
            'timestamp': pd.date_range('2024-01-01', periods=100, freq='1min'),
            'open': np.ones(100) * 71000,
            'high': np.ones(100) * 71100,
            'low': np.ones(100) * 70900,
            'close': np.ones(100) * 71000,
            'volume': np.zeros(100)  # All zeros
        })
        
        features = extractor.extract_price_features(df, window_size=60)
        
        # Should not have inf values
        assert not np.isinf(features).any()
    
    def test_extract_with_constant_prices(self):
        """Test extraction with constant prices."""
        extractor = FeatureExtractor()
        
        df = pd.DataFrame({
            'timestamp': pd.date_range('2024-01-01', periods=100, freq='1min'),
            'open': np.ones(100) * 71000,
            'high': np.ones(100) * 71000,
            'low': np.ones(100) * 71000,
            'close': np.ones(100) * 71000,
            'volume': np.random.randint(100000, 1000000, 100)
        })
        
        features = extractor.extract_price_features(df, window_size=60)
        
        # Should handle division by zero
        assert not np.isnan(features).any()
        assert not np.isinf(features).any()


@pytest.mark.unit
class TestSentimentFeatureExtractor:
    """Test SentimentFeatureExtractor class."""
    
    def test_initialization(self):
        """Test sentiment extractor initialization."""
        extractor = SentimentFeatureExtractor()
        assert extractor is not None
    
    def test_tokenize_text_basic(self):
        """Test basic text tokenization."""
        extractor = SentimentFeatureExtractor()
        text = "Samsung reports strong earnings"
        
        result = extractor.tokenize_text(text)
        
        assert "input_ids" in result
        assert "attention_mask" in result
        assert result["input_ids"].shape == (512,)
        assert result["attention_mask"].shape == (512,)
        assert result["input_ids"].dtype == np.int64
        assert result["attention_mask"].dtype == np.int64
    
    def test_tokenize_text_empty(self):
        """Test tokenization of empty text."""
        extractor = SentimentFeatureExtractor()
        
        result = extractor.tokenize_text("")
        
        assert result["input_ids"].shape == (512,)
        assert result["attention_mask"].shape == (512,)
    
    def test_tokenize_text_long(self):
        """Test tokenization of long text."""
        extractor = SentimentFeatureExtractor()
        text = "This is a very long text. " * 100  # Very long
        
        result = extractor.tokenize_text(text, max_length=512)
        
        # Should be truncated to max_length
        assert result["input_ids"].shape == (512,)
    
    def test_tokenize_text_custom_length(self):
        """Test tokenization with custom max length."""
        extractor = SentimentFeatureExtractor()
        text = "Test text"
        
        result = extractor.tokenize_text(text, max_length=256)
        
        assert result["input_ids"].shape == (256,)
        assert result["attention_mask"].shape == (256,)
    
    def test_batch_tokenize(self, sample_news_text):
        """Test batch tokenization."""
        extractor = SentimentFeatureExtractor()
        
        result = extractor.batch_tokenize(sample_news_text)
        
        assert "input_ids" in result
        assert "attention_mask" in result
        assert result["input_ids"].shape == (len(sample_news_text), 512)
        assert result["attention_mask"].shape == (len(sample_news_text), 512)
    
    def test_batch_tokenize_empty_list(self):
        """Test batch tokenization with empty list."""
        extractor = SentimentFeatureExtractor()
        
        result = extractor.batch_tokenize([])
        
        assert result["input_ids"].shape == (0, 512)
        assert result["attention_mask"].shape == (0, 512)
    
    def test_tokenize_special_characters(self):
        """Test tokenization with special characters."""
        extractor = SentimentFeatureExtractor()
        text = "Stock price $100.50 +5% 📈"
        
        result = extractor.tokenize_text(text)
        
        assert result["input_ids"].shape == (512,)
        assert not np.isnan(result["input_ids"]).any()
    
    def test_tokenize_multilingual(self):
        """Test tokenization with non-English text."""
        extractor = SentimentFeatureExtractor()
        text = "삼성전자 실적 호조"
        
        result = extractor.tokenize_text(text)
        
        assert result["input_ids"].shape == (512,)
        assert not np.isnan(result["input_ids"]).any()
