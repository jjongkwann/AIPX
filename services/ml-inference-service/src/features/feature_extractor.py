"""Feature extraction for ML models."""

import logging
from typing import Dict, Optional

import numpy as np
import pandas as pd
from sklearn.preprocessing import MinMaxScaler, StandardScaler

from ..config import settings

logger = logging.getLogger(__name__)


class FeatureExtractor:
    """Extract and normalize features for ML models."""

    def __init__(self, normalization: str = "standard"):
        """
        Initialize feature extractor.

        Args:
            normalization: Normalization method ('standard', 'minmax', 'none')
        """
        self.normalization = normalization
        self.scaler = None

        if normalization == "standard":
            self.scaler = StandardScaler()
        elif normalization == "minmax":
            self.scaler = MinMaxScaler()

    def extract_price_features(self, df: pd.DataFrame, window_size: int = 60) -> np.ndarray:
        """
        Extract features for LSTM price prediction.

        Args:
            df: DataFrame with OHLCV columns
            window_size: Number of timesteps to extract

        Returns:
            features: shape (window_size, 5)
        """
        try:
            # Validate input
            required_columns = ["open", "high", "low", "close", "volume"]
            if not all(col in df.columns for col in required_columns):
                raise ValueError(f"DataFrame must contain columns: {required_columns}")

            if len(df) < window_size:
                raise ValueError(f"DataFrame has {len(df)} rows, need at least {window_size}")

            # Get last window_size rows
            window_df = df.tail(window_size).copy()

            # Extract base prices
            close_prices = window_df["close"].values
            open_prices = window_df["open"].values
            high_prices = window_df["high"].values
            low_prices = window_df["low"].values
            volumes = window_df["volume"].values

            # Calculate price ratios (relative to close)
            open_close_ratio = open_prices / close_prices
            high_close_ratio = high_prices / close_prices
            low_close_ratio = low_prices / close_prices

            # Normalize close prices
            close_normalized = (close_prices - close_prices.mean()) / (close_prices.std() + 1e-8)

            # Normalize volume
            volume_normalized = (volumes - volumes.mean()) / (volumes.std() + 1e-8)

            # Stack features
            features = np.column_stack(
                [close_normalized, open_close_ratio, high_close_ratio, low_close_ratio, volume_normalized]
            )

            return features.astype(np.float32)

        except Exception as e:
            logger.error(f"Feature extraction error: {e}")
            raise

    def extract_technical_indicators(self, df: pd.DataFrame, window_size: int = 60) -> np.ndarray:
        """
        Extract technical indicators as additional features.

        Args:
            df: DataFrame with OHLCV columns
            window_size: Number of timesteps

        Returns:
            indicators: shape (window_size, n_indicators)
        """
        try:
            window_df = df.tail(window_size).copy()

            indicators = []

            # Returns
            returns = window_df["close"].pct_change().fillna(0).values
            indicators.append(returns)

            # Moving averages (if enough data)
            if len(df) >= window_size + 20:
                ma_5 = window_df["close"].rolling(5).mean().fillna(method="bfill").values
                ma_20 = window_df["close"].rolling(20).mean().fillna(method="bfill").values
                ma_ratio = ma_5 / (ma_20 + 1e-8)
                indicators.append(ma_ratio)

            # RSI (Relative Strength Index)
            delta = window_df["close"].diff()
            gain = (delta.where(delta > 0, 0)).rolling(14).mean()
            loss = (-delta.where(delta < 0, 0)).rolling(14).mean()
            rs = gain / (loss + 1e-8)
            rsi = 100 - (100 / (1 + rs))
            indicators.append(rsi.fillna(50).values / 100.0)  # Normalize to [0, 1]

            # Volume ratio
            volume_ma = window_df["volume"].rolling(20).mean().fillna(method="bfill")
            volume_ratio = window_df["volume"] / (volume_ma + 1e-8)
            indicators.append(volume_ratio.values)

            # Volatility (standard deviation of returns)
            volatility = window_df["close"].pct_change().rolling(20).std().fillna(0).values
            indicators.append(volatility)

            # Stack indicators
            features = np.column_stack(indicators)

            return features.astype(np.float32)

        except Exception as e:
            logger.error(f"Technical indicator extraction error: {e}")
            raise

    def extract_advanced_features(self, df: pd.DataFrame, window_size: int = 60) -> np.ndarray:
        """
        Extract advanced features combining price and technical indicators.

        Args:
            df: DataFrame with OHLCV columns
            window_size: Number of timesteps

        Returns:
            features: shape (window_size, total_features)
        """
        try:
            # Extract base price features
            price_features = self.extract_price_features(df, window_size)

            # Extract technical indicators
            tech_features = self.extract_technical_indicators(df, window_size)

            # Combine all features
            all_features = np.concatenate([price_features, tech_features], axis=1)

            return all_features

        except Exception as e:
            logger.error(f"Advanced feature extraction error: {e}")
            raise

    def normalize_features(self, features: np.ndarray) -> np.ndarray:
        """
        Normalize features using configured scaler.

        Args:
            features: Input features

        Returns:
            Normalized features
        """
        if self.scaler is None:
            return features

        try:
            # Fit and transform
            original_shape = features.shape
            features_2d = features.reshape(-1, features.shape[-1])
            normalized = self.scaler.fit_transform(features_2d)
            return normalized.reshape(original_shape)

        except Exception as e:
            logger.error(f"Normalization error: {e}")
            raise

    def prepare_lstm_input(self, df: pd.DataFrame, window_size: Optional[int] = None) -> np.ndarray:
        """
        Prepare input features for LSTM model.

        Args:
            df: DataFrame with OHLCV data
            window_size: Window size (default: from settings)

        Returns:
            features: shape (window_size, 5) ready for LSTM
        """
        window_size = window_size or settings.feature_window_size

        # Extract features
        features = self.extract_price_features(df, window_size)

        # Validate output shape
        if features.shape != (window_size, 5):
            raise ValueError(f"Expected shape ({window_size}, 5), got {features.shape}")

        return features

    def prepare_batch_lstm_input(self, dfs: list[pd.DataFrame], window_size: Optional[int] = None) -> np.ndarray:
        """
        Prepare batch input for LSTM model.

        Args:
            dfs: List of DataFrames
            window_size: Window size

        Returns:
            features: shape (batch_size, window_size, 5)
        """
        window_size = window_size or settings.feature_window_size

        batch_features = []
        for df in dfs:
            features = self.prepare_lstm_input(df, window_size)
            batch_features.append(features)

        return np.stack(batch_features, axis=0)

    def calculate_feature_importance(self, features: np.ndarray, predictions: np.ndarray) -> Dict[str, float]:
        """
        Calculate feature importance scores.

        Args:
            features: Input features
            predictions: Model predictions

        Returns:
            Dictionary of feature importance scores
        """
        try:
            # Simple correlation-based importance
            feature_names = [
                "close_normalized",
                "open_close_ratio",
                "high_close_ratio",
                "low_close_ratio",
                "volume_normalized",
            ]

            importance = {}
            pred_flat = predictions.flatten()
            for i, name in enumerate(feature_names):
                # Use last timestep features (most recent) for correlation with predictions
                feature_values = features[:, -1, i]
                if len(feature_values) == len(pred_flat):
                    correlation = np.corrcoef(feature_values, pred_flat)[0, 1]
                    if not np.isnan(correlation):
                        importance[name] = abs(correlation)
                    else:
                        importance[name] = 0.0
                else:
                    importance[name] = 0.0

            return importance

        except Exception as e:
            logger.error(f"Feature importance calculation error: {e}")
            return {}


class SentimentFeatureExtractor:
    """Extract features for sentiment analysis."""

    def __init__(self, tokenizer_name: str = "bert-base-uncased"):
        """
        Initialize sentiment feature extractor.

        Args:
            tokenizer_name: Name of the tokenizer to use
        """
        try:
            from transformers import AutoTokenizer

            self.tokenizer = AutoTokenizer.from_pretrained(tokenizer_name)
        except Exception as e:
            logger.warning(f"Failed to load tokenizer: {e}. Using dummy tokenizer.")
            self.tokenizer = None

    def tokenize_text(self, text: str, max_length: int = 512) -> Dict[str, np.ndarray]:
        """
        Tokenize text for transformer model.

        Args:
            text: Input text
            max_length: Maximum sequence length

        Returns:
            Dictionary with input_ids and attention_mask
        """
        if self.tokenizer is None:
            # Dummy tokenization for testing
            input_ids = np.zeros(max_length, dtype=np.int64)
            attention_mask = np.ones(max_length, dtype=np.int64)
            return {"input_ids": input_ids, "attention_mask": attention_mask}

        try:
            # Tokenize
            encoded = self.tokenizer(
                text, max_length=max_length, padding="max_length", truncation=True, return_tensors="np"
            )

            return {
                "input_ids": encoded["input_ids"][0].astype(np.int64),
                "attention_mask": encoded["attention_mask"][0].astype(np.int64),
            }

        except Exception as e:
            logger.error(f"Tokenization error: {e}")
            raise

    def batch_tokenize(self, texts: list[str], max_length: int = 512) -> Dict[str, np.ndarray]:
        """
        Batch tokenize multiple texts.

        Args:
            texts: List of input texts
            max_length: Maximum sequence length

        Returns:
            Dictionary with batched input_ids and attention_mask
        """
        if not texts:
            return {
                "input_ids": np.zeros((0, max_length), dtype=np.int64),
                "attention_mask": np.zeros((0, max_length), dtype=np.int64),
            }

        batch_tokens = [self.tokenize_text(text, max_length) for text in texts]

        return {
            "input_ids": np.stack([t["input_ids"] for t in batch_tokens]),
            "attention_mask": np.stack([t["attention_mask"] for t in batch_tokens]),
        }


# Global feature extractors
price_feature_extractor = FeatureExtractor(normalization=settings.feature_normalization)
sentiment_feature_extractor = SentimentFeatureExtractor()
