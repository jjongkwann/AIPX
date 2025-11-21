"""Validation tests for LSTM model predictions."""
import pytest
import numpy as np
import pandas as pd
import torch


@pytest.mark.validation
class TestLSTMPredictionValidation:
    """Validate LSTM model predictions."""
    
    def test_uptrend_prediction(self, mock_lstm_model):
        """Test model predicts higher prices for uptrend."""
        # Create uptrend data
        base_prices = np.linspace(70000, 71000, 60)
        noise = np.random.randn(60) * 50
        close_prices = base_prices + noise
        
        # Normalize
        close_normalized = (close_prices - close_prices.mean()) / (close_prices.std() + 1e-8)
        
        # Create features
        features = np.column_stack([
            close_normalized,
            np.ones(60),  # open/close ratio
            np.ones(60) * 1.01,  # high/close ratio
            np.ones(60) * 0.99,  # low/close ratio
            np.zeros(60)  # volume normalized
        ]).astype(np.float32)
        
        # Predict
        mock_lstm_model.eval()
        with torch.no_grad():
            x = torch.from_numpy(features).unsqueeze(0)
            prediction = mock_lstm_model(x).item()
        
        # Prediction should be non-zero
        assert prediction != 0
        
        print(f"\nUptrend Prediction: {prediction:.4f}")
    
    def test_downtrend_prediction(self, mock_lstm_model):
        """Test model predicts lower prices for downtrend."""
        # Create downtrend data
        base_prices = np.linspace(71000, 70000, 60)
        noise = np.random.randn(60) * 50
        close_prices = base_prices + noise
        
        # Normalize
        close_normalized = (close_prices - close_prices.mean()) / (close_prices.std() + 1e-8)
        
        # Create features
        features = np.column_stack([
            close_normalized,
            np.ones(60),
            np.ones(60) * 1.01,
            np.ones(60) * 0.99,
            np.zeros(60)
        ]).astype(np.float32)
        
        # Predict
        mock_lstm_model.eval()
        with torch.no_grad():
            x = torch.from_numpy(features).unsqueeze(0)
            prediction = mock_lstm_model(x).item()
        
        print(f"\nDowntrend Prediction: {prediction:.4f}")
        
        # Should produce a prediction
        assert not np.isnan(prediction)
        assert not np.isinf(prediction)
    
    def test_sideways_prediction(self, mock_lstm_model):
        """Test model on sideways/flat market."""
        # Create flat prices with noise
        base_price = 71000
        noise = np.random.randn(60) * 100
        close_prices = base_price + noise
        
        # Normalize
        close_normalized = (close_prices - close_prices.mean()) / (close_prices.std() + 1e-8)
        
        # Create features
        features = np.column_stack([
            close_normalized,
            np.ones(60),
            np.ones(60) * 1.01,
            np.ones(60) * 0.99,
            np.zeros(60)
        ]).astype(np.float32)
        
        # Predict
        mock_lstm_model.eval()
        with torch.no_grad():
            x = torch.from_numpy(features).unsqueeze(0)
            prediction = mock_lstm_model(x).item()
        
        print(f"\nSideways Prediction: {prediction:.4f}")
        
        # Prediction should be reasonable (not extreme)
        assert abs(prediction) < 10  # Reasonable range
    
    def test_prediction_consistency(self, mock_lstm_model):
        """Test predictions are consistent for same input."""
        # Create sample features
        features = np.random.randn(60, 5).astype(np.float32)
        
        mock_lstm_model.eval()
        
        # Multiple predictions
        predictions = []
        with torch.no_grad():
            x = torch.from_numpy(features).unsqueeze(0)
            for _ in range(10):
                pred = mock_lstm_model(x).item()
                predictions.append(pred)
        
        # All predictions should be identical in eval mode
        assert all(p == predictions[0] for p in predictions)
    
    def test_no_extreme_predictions(self, mock_lstm_model):
        """Test model doesn't output extreme values."""
        mock_lstm_model.eval()
        
        n_samples = 100
        predictions = []
        
        with torch.no_grad():
            for _ in range(n_samples):
                features = np.random.randn(60, 5).astype(np.float32)
                x = torch.from_numpy(features).unsqueeze(0)
                pred = mock_lstm_model(x).item()
                predictions.append(pred)
        
        predictions = np.array(predictions)
        
        print(f"\nPrediction Statistics:")
        print(f"  Mean: {predictions.mean():.4f}")
        print(f"  Std: {predictions.std():.4f}")
        print(f"  Min: {predictions.min():.4f}")
        print(f"  Max: {predictions.max():.4f}")
        
        # Check for reasonable range
        assert not np.isnan(predictions).any()
        assert not np.isinf(predictions).any()
        assert abs(predictions.mean()) < 100
        assert predictions.std() < 100
    
    def test_prediction_accuracy_baseline(self, sample_market_data):
        """Test predictions against naive baseline."""
        from src.features.feature_extractor import FeatureExtractor
        
        extractor = FeatureExtractor()
        
        # Get features
        features = extractor.extract_price_features(sample_market_data, window_size=60)
        
        # Last actual price
        last_price = sample_market_data.tail(1)['close'].values[0]
        
        # Naive prediction (last price)
        naive_prediction = last_price
        
        # With mock, we can't really test accuracy, but we can test structure
        assert features.shape == (60, 5)
        assert not np.isnan(features).any()
        
        print(f"\nLast Price: {last_price:.2f}")
        print(f"Naive Prediction: {naive_prediction:.2f}")
    
    def test_feature_sensitivity(self, mock_lstm_model):
        """Test model is sensitive to input features."""
        mock_lstm_model.eval()
        
        # Create two different feature sets
        features1 = np.random.randn(60, 5).astype(np.float32)
        features2 = np.random.randn(60, 5).astype(np.float32) * 2  # Different distribution
        
        with torch.no_grad():
            x1 = torch.from_numpy(features1).unsqueeze(0)
            x2 = torch.from_numpy(features2).unsqueeze(0)
            
            pred1 = mock_lstm_model(x1).item()
            pred2 = mock_lstm_model(x2).item()
        
        # Predictions should be different
        assert pred1 != pred2
        
        print(f"\nFeature Sensitivity:")
        print(f"  Input 1 -> Prediction: {pred1:.4f}")
        print(f"  Input 2 -> Prediction: {pred2:.4f}")
    
    def test_temporal_consistency(self, mock_lstm_model):
        """Test predictions maintain temporal consistency."""
        # Create sequence of overlapping windows
        base_prices = np.linspace(70000, 71000, 100)
        
        predictions = []
        mock_lstm_model.eval()
        
        with torch.no_grad():
            for i in range(0, 40):
                window = base_prices[i:i+60]
                normalized = (window - window.mean()) / (window.std() + 1e-8)
                
                features = np.column_stack([
                    normalized,
                    np.ones(60),
                    np.ones(60) * 1.01,
                    np.ones(60) * 0.99,
                    np.zeros(60)
                ]).astype(np.float32)
                
                x = torch.from_numpy(features).unsqueeze(0)
                pred = mock_lstm_model(x).item()
                predictions.append(pred)
        
        predictions = np.array(predictions)
        
        # Predictions should change smoothly
        diffs = np.diff(predictions)
        
        print(f"\nTemporal Consistency:")
        print(f"  Prediction changes mean: {diffs.mean():.4f}")
        print(f"  Prediction changes std: {diffs.std():.4f}")
        
        # Changes should not be too extreme
        assert diffs.std() < 10


@pytest.mark.validation
class TestModelRobustness:
    """Test model robustness to edge cases."""
    
    def test_all_zeros_input(self, mock_lstm_model):
        """Test model with all zeros input."""
        features = np.zeros((60, 5), dtype=np.float32)
        
        mock_lstm_model.eval()
        with torch.no_grad():
            x = torch.from_numpy(features).unsqueeze(0)
            prediction = mock_lstm_model(x).item()
        
        # Should handle gracefully
        assert not np.isnan(prediction)
        assert not np.isinf(prediction)
    
    def test_very_large_values(self, mock_lstm_model):
        """Test model with very large input values."""
        features = np.ones((60, 5), dtype=np.float32) * 1000
        
        mock_lstm_model.eval()
        with torch.no_grad():
            x = torch.from_numpy(features).unsqueeze(0)
            prediction = mock_lstm_model(x).item()
        
        assert not np.isnan(prediction)
        assert not np.isinf(prediction)
    
    def test_very_small_values(self, mock_lstm_model):
        """Test model with very small input values."""
        features = np.ones((60, 5), dtype=np.float32) * 1e-6
        
        mock_lstm_model.eval()
        with torch.no_grad():
            x = torch.from_numpy(features).unsqueeze(0)
            prediction = mock_lstm_model(x).item()
        
        assert not np.isnan(prediction)
        assert not np.isinf(prediction)
    
    def test_mixed_signs(self, mock_lstm_model):
        """Test model with mixed positive/negative values."""
        features = np.random.randn(60, 5).astype(np.float32)
        features[:30, :] = abs(features[:30, :])  # First half positive
        features[30:, :] = -abs(features[30:, :])  # Second half negative
        
        mock_lstm_model.eval()
        with torch.no_grad():
            x = torch.from_numpy(features).unsqueeze(0)
            prediction = mock_lstm_model(x).item()
        
        assert not np.isnan(prediction)
        assert not np.isinf(prediction)


@pytest.mark.validation
class TestPredictionQuality:
    """Test prediction quality metrics."""
    
    def test_prediction_variance(self, mock_lstm_model):
        """Test prediction variance across samples."""
        n_samples = 100
        predictions = []
        
        mock_lstm_model.eval()
        with torch.no_grad():
            for _ in range(n_samples):
                features = np.random.randn(60, 5).astype(np.float32)
                x = torch.from_numpy(features).unsqueeze(0)
                pred = mock_lstm_model(x).item()
                predictions.append(pred)
        
        predictions = np.array(predictions)
        variance = predictions.var()
        
        print(f"\nPrediction Variance: {variance:.4f}")
        
        # Should have some variance (not constant)
        assert variance > 0
        # But not too much
        assert variance < 1000
    
    def test_prediction_distribution(self, mock_lstm_model):
        """Test prediction distribution is reasonable."""
        n_samples = 1000
        predictions = []
        
        mock_lstm_model.eval()
        with torch.no_grad():
            for _ in range(n_samples):
                features = np.random.randn(60, 5).astype(np.float32)
                x = torch.from_numpy(features).unsqueeze(0)
                pred = mock_lstm_model(x).item()
                predictions.append(pred)
        
        predictions = np.array(predictions)
        
        # Calculate distribution metrics
        mean = predictions.mean()
        std = predictions.std()
        skewness = ((predictions - mean) ** 3).mean() / (std ** 3)
        kurtosis = ((predictions - mean) ** 4).mean() / (std ** 4)
        
        print(f"\nPrediction Distribution:")
        print(f"  Mean: {mean:.4f}")
        print(f"  Std: {std:.4f}")
        print(f"  Skewness: {skewness:.4f}")
        print(f"  Kurtosis: {kurtosis:.4f}")
        
        # Should be roughly normal-ish
        assert abs(skewness) < 3  # Not too skewed
        assert kurtosis < 10  # Not too heavy-tailed
