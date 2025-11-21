"""Data loader for historical market data"""

import pandas as pd
from datetime import datetime
from typing import Optional, List


class DataLoader:
    """Load historical tick data for backtesting"""

    def __init__(self, data_source: Optional[any] = None):
        """
        Initialize data loader

        Args:
            data_source: Database connection or data source (optional for testing)
        """
        self.data_source = data_source

    def load_tick_data(
        self,
        symbol: str,
        start_date: datetime,
        end_date: datetime
    ) -> pd.DataFrame:
        """
        Load tick data for a symbol within date range

        Args:
            symbol: Stock symbol (e.g., '005930')
            start_date: Start datetime
            end_date: End datetime

        Returns:
            DataFrame with columns: timestamp, symbol, open, high, low, close, volume
        """
        if not symbol:
            raise ValueError("symbol cannot be empty")

        if start_date >= end_date:
            raise ValueError("start_date must be before end_date")

        # If no data source, return empty DataFrame
        if self.data_source is None:
            return pd.DataFrame(columns=[
                'timestamp', 'symbol', 'open', 'high', 'low', 'close', 'volume'
            ])

        # In production, this would query the database
        # For now, return empty DataFrame
        return pd.DataFrame(columns=[
            'timestamp', 'symbol', 'open', 'high', 'low', 'close', 'volume'
        ])

    def load_multiple_symbols(
        self,
        symbols: List[str],
        start_date: datetime,
        end_date: datetime
    ) -> pd.DataFrame:
        """
        Load tick data for multiple symbols

        Args:
            symbols: List of stock symbols
            start_date: Start datetime
            end_date: End datetime

        Returns:
            DataFrame with all symbols' data concatenated
        """
        if not symbols:
            raise ValueError("symbols list cannot be empty")

        all_data = []
        for symbol in symbols:
            data = self.load_tick_data(symbol, start_date, end_date)
            all_data.append(data)

        if not all_data:
            return pd.DataFrame(columns=[
                'timestamp', 'symbol', 'open', 'high', 'low', 'close', 'volume'
            ])

        return pd.concat(all_data, ignore_index=True)

    def validate_data(self, data: pd.DataFrame) -> bool:
        """
        Validate that data has required columns and valid values

        Args:
            data: DataFrame to validate

        Returns:
            True if valid, False otherwise
        """
        required_columns = ['timestamp', 'symbol', 'open', 'high', 'low', 'close', 'volume']

        # Check columns exist
        if not all(col in data.columns for col in required_columns):
            return False

        # Check no null values in critical columns
        if data[required_columns].isnull().any().any():
            return False

        # Check positive prices and volumes
        if (data['open'] <= 0).any() or (data['high'] <= 0).any() or \
           (data['low'] <= 0).any() or (data['close'] <= 0).any():
            return False

        if (data['volume'] < 0).any():
            return False

        # Check high >= low
        if (data['high'] < data['low']).any():
            return False

        return True
