"""Unit tests for Data Loader"""

import pytest
import pandas as pd
from datetime import datetime
from src.data.data_loader import DataLoader


class TestDataLoader:
    """Test DataLoader class"""

    def test_initialization(self):
        """Test data loader initialization"""
        loader = DataLoader()

        assert loader.data_source is None

    def test_initialization_with_source(self):
        """Test initialization with data source"""
        mock_source = "mock_db"
        loader = DataLoader(data_source=mock_source)

        assert loader.data_source == mock_source

    def test_load_tick_data_returns_dataframe(self):
        """Test that load_tick_data returns a DataFrame"""
        loader = DataLoader()

        data = loader.load_tick_data(
            symbol='005930',
            start_date=datetime(2024, 1, 1),
            end_date=datetime(2024, 12, 31)
        )

        assert isinstance(data, pd.DataFrame)

    def test_load_tick_data_has_correct_columns(self):
        """Test that returned DataFrame has correct columns"""
        loader = DataLoader()

        data = loader.load_tick_data(
            symbol='005930',
            start_date=datetime(2024, 1, 1),
            end_date=datetime(2024, 12, 31)
        )

        expected_columns = ['timestamp', 'symbol', 'open', 'high', 'low', 'close', 'volume']
        assert all(col in data.columns for col in expected_columns)

    def test_load_tick_data_empty_symbol(self):
        """Test that empty symbol raises error"""
        loader = DataLoader()

        with pytest.raises(ValueError, match="symbol cannot be empty"):
            loader.load_tick_data(
                symbol='',
                start_date=datetime(2024, 1, 1),
                end_date=datetime(2024, 12, 31)
            )

    def test_load_tick_data_invalid_date_range(self):
        """Test that invalid date range raises error"""
        loader = DataLoader()

        with pytest.raises(ValueError, match="start_date must be before end_date"):
            loader.load_tick_data(
                symbol='005930',
                start_date=datetime(2024, 12, 31),
                end_date=datetime(2024, 1, 1)
            )

    def test_load_tick_data_same_dates(self):
        """Test that same start and end date raises error"""
        loader = DataLoader()

        with pytest.raises(ValueError):
            loader.load_tick_data(
                symbol='005930',
                start_date=datetime(2024, 1, 1),
                end_date=datetime(2024, 1, 1)
            )

    def test_load_multiple_symbols(self):
        """Test loading data for multiple symbols"""
        loader = DataLoader()

        data = loader.load_multiple_symbols(
            symbols=['005930', '000660'],
            start_date=datetime(2024, 1, 1),
            end_date=datetime(2024, 12, 31)
        )

        assert isinstance(data, pd.DataFrame)

    def test_load_multiple_symbols_empty_list(self):
        """Test that empty symbols list raises error"""
        loader = DataLoader()

        with pytest.raises(ValueError, match="symbols list cannot be empty"):
            loader.load_multiple_symbols(
                symbols=[],
                start_date=datetime(2024, 1, 1),
                end_date=datetime(2024, 12, 31)
            )

    def test_validate_data_valid(self, sample_tick_data):
        """Test validation with valid data"""
        loader = DataLoader()

        is_valid = loader.validate_data(sample_tick_data)

        assert is_valid is True

    def test_validate_data_missing_columns(self):
        """Test validation with missing columns"""
        loader = DataLoader()

        # Missing 'volume' column
        data = pd.DataFrame({
            'timestamp': [datetime(2024, 1, 1)],
            'symbol': ['005930'],
            'open': [71000],
            'high': [72000],
            'low': [70000],
            'close': [71500]
        })

        is_valid = loader.validate_data(data)

        assert is_valid is False

    def test_validate_data_null_values(self):
        """Test validation with null values"""
        loader = DataLoader()

        data = pd.DataFrame({
            'timestamp': [datetime(2024, 1, 1), datetime(2024, 1, 2)],
            'symbol': ['005930', '005930'],
            'open': [71000, None],  # Null value
            'high': [72000, 72500],
            'low': [70000, 71000],
            'close': [71500, 72000],
            'volume': [1000000, 1100000]
        })

        is_valid = loader.validate_data(data)

        assert is_valid is False

    def test_validate_data_negative_prices(self):
        """Test validation with negative prices"""
        loader = DataLoader()

        data = pd.DataFrame({
            'timestamp': [datetime(2024, 1, 1)],
            'symbol': ['005930'],
            'open': [-71000],  # Negative price
            'high': [72000],
            'low': [70000],
            'close': [71500],
            'volume': [1000000]
        })

        is_valid = loader.validate_data(data)

        assert is_valid is False

    def test_validate_data_zero_prices(self):
        """Test validation with zero prices"""
        loader = DataLoader()

        data = pd.DataFrame({
            'timestamp': [datetime(2024, 1, 1)],
            'symbol': ['005930'],
            'open': [0],  # Zero price
            'high': [72000],
            'low': [70000],
            'close': [71500],
            'volume': [1000000]
        })

        is_valid = loader.validate_data(data)

        assert is_valid is False

    def test_validate_data_negative_volume(self):
        """Test validation with negative volume"""
        loader = DataLoader()

        data = pd.DataFrame({
            'timestamp': [datetime(2024, 1, 1)],
            'symbol': ['005930'],
            'open': [71000],
            'high': [72000],
            'low': [70000],
            'close': [71500],
            'volume': [-1000000]  # Negative volume
        })

        is_valid = loader.validate_data(data)

        assert is_valid is False

    def test_validate_data_high_less_than_low(self):
        """Test validation when high < low"""
        loader = DataLoader()

        data = pd.DataFrame({
            'timestamp': [datetime(2024, 1, 1)],
            'symbol': ['005930'],
            'open': [71000],
            'high': [70000],  # High < Low
            'low': [72000],
            'close': [71500],
            'volume': [1000000]
        })

        is_valid = loader.validate_data(data)

        assert is_valid is False

    def test_validate_data_empty_dataframe(self):
        """Test validation with empty DataFrame"""
        loader = DataLoader()

        data = pd.DataFrame(columns=[
            'timestamp', 'symbol', 'open', 'high', 'low', 'close', 'volume'
        ])

        # Empty data is technically valid (no violations)
        is_valid = loader.validate_data(data)

        assert is_valid is True

    def test_validate_data_realistic_ohlc(self):
        """Test validation with realistic OHLC data"""
        loader = DataLoader()

        data = pd.DataFrame({
            'timestamp': [
                datetime(2024, 1, 1),
                datetime(2024, 1, 2),
                datetime(2024, 1, 3)
            ],
            'symbol': ['005930', '005930', '005930'],
            'open': [71000, 71500, 72000],
            'high': [72500, 72800, 73000],
            'low': [70500, 71000, 71500],
            'close': [71500, 72000, 72500],
            'volume': [5000000, 5500000, 6000000]
        })

        is_valid = loader.validate_data(data)

        assert is_valid is True

    def test_load_tick_data_integration_with_validation(self, sample_tick_data):
        """Test that loaded data passes validation"""
        # This would test with actual database in production
        loader = DataLoader()

        # Use fixture data
        is_valid = loader.validate_data(sample_tick_data)

        assert is_valid is True

    def test_load_multiple_symbols_returns_correct_structure(self):
        """Test that multiple symbols return correct DataFrame structure"""
        loader = DataLoader()

        data = loader.load_multiple_symbols(
            symbols=['005930', '000660', '035720'],
            start_date=datetime(2024, 1, 1),
            end_date=datetime(2024, 12, 31)
        )

        # Should have all required columns
        expected_columns = ['timestamp', 'symbol', 'open', 'high', 'low', 'close', 'volume']
        assert all(col in data.columns for col in expected_columns)
