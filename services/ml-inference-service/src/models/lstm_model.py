"""LSTM model for price prediction."""
import torch
import torch.nn as nn
from typing import Tuple, Optional
import logging

logger = logging.getLogger(__name__)


class LSTMPredictor(nn.Module):
    """LSTM model for next-tick price prediction."""

    def __init__(
        self,
        input_size: int = 5,
        hidden_size: int = 128,
        num_layers: int = 2,
        dropout: float = 0.2,
        bidirectional: bool = False
    ):
        """
        Initialize LSTM predictor.

        Args:
            input_size: Number of input features
            hidden_size: Hidden layer size
            num_layers: Number of LSTM layers
            dropout: Dropout rate
            bidirectional: Use bidirectional LSTM
        """
        super(LSTMPredictor, self).__init__()

        self.input_size = input_size
        self.hidden_size = hidden_size
        self.num_layers = num_layers
        self.bidirectional = bidirectional

        # LSTM layer
        self.lstm = nn.LSTM(
            input_size=input_size,
            hidden_size=hidden_size,
            num_layers=num_layers,
            batch_first=True,
            dropout=dropout if num_layers > 1 else 0,
            bidirectional=bidirectional
        )

        # Attention layer
        self.attention = nn.Linear(
            hidden_size * (2 if bidirectional else 1),
            1
        )

        # Output layers
        fc_input_size = hidden_size * (2 if bidirectional else 1)
        self.fc1 = nn.Linear(fc_input_size, hidden_size // 2)
        self.relu = nn.ReLU()
        self.dropout = nn.Dropout(dropout)
        self.fc2 = nn.Linear(hidden_size // 2, 1)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """
        Forward pass.

        Args:
            x: Input tensor of shape (batch, seq_len, input_size)

        Returns:
            predictions: Output tensor of shape (batch, 1)
        """
        # LSTM forward pass
        # output: (batch, seq_len, hidden_size * num_directions)
        # hidden: (num_layers * num_directions, batch, hidden_size)
        output, (hidden, cell) = self.lstm(x)

        # Apply attention
        attention_weights = torch.softmax(self.attention(output), dim=1)
        context = torch.sum(attention_weights * output, dim=1)

        # Fully connected layers
        out = self.fc1(context)
        out = self.relu(out)
        out = self.dropout(out)
        predictions = self.fc2(out)

        return predictions

    def get_attention_weights(self, x: torch.Tensor) -> torch.Tensor:
        """
        Get attention weights for interpretability.

        Args:
            x: Input tensor

        Returns:
            Attention weights
        """
        with torch.no_grad():
            output, _ = self.lstm(x)
            attention_weights = torch.softmax(self.attention(output), dim=1)
            return attention_weights


class AdvancedLSTMPredictor(nn.Module):
    """Advanced LSTM with residual connections and layer normalization."""

    def __init__(
        self,
        input_size: int = 5,
        hidden_size: int = 256,
        num_layers: int = 3,
        dropout: float = 0.3
    ):
        """Initialize advanced LSTM predictor."""
        super(AdvancedLSTMPredictor, self).__init__()

        self.input_size = input_size
        self.hidden_size = hidden_size
        self.num_layers = num_layers

        # Input projection
        self.input_projection = nn.Linear(input_size, hidden_size)
        self.layer_norm_input = nn.LayerNorm(hidden_size)

        # LSTM layers with residual connections
        self.lstm_layers = nn.ModuleList([
            nn.LSTM(
                input_size=hidden_size,
                hidden_size=hidden_size,
                num_layers=1,
                batch_first=True,
                dropout=0
            )
            for _ in range(num_layers)
        ])

        # Layer normalization for each LSTM layer
        self.layer_norms = nn.ModuleList([
            nn.LayerNorm(hidden_size)
            for _ in range(num_layers)
        ])

        # Dropout
        self.dropout = nn.Dropout(dropout)

        # Multi-head attention
        self.multihead_attention = nn.MultiheadAttention(
            embed_dim=hidden_size,
            num_heads=8,
            dropout=dropout,
            batch_first=True
        )

        # Output layers
        self.fc1 = nn.Linear(hidden_size, hidden_size // 2)
        self.fc2 = nn.Linear(hidden_size // 2, hidden_size // 4)
        self.fc3 = nn.Linear(hidden_size // 4, 1)
        self.relu = nn.ReLU()

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """
        Forward pass with residual connections.

        Args:
            x: Input tensor (batch, seq_len, input_size)

        Returns:
            predictions: (batch, 1)
        """
        # Input projection
        out = self.input_projection(x)
        out = self.layer_norm_input(out)

        # LSTM layers with residual connections
        for i, (lstm, layer_norm) in enumerate(zip(self.lstm_layers, self.layer_norms)):
            residual = out
            out, _ = lstm(out)
            out = layer_norm(out)
            out = self.dropout(out)

            # Add residual connection
            if i > 0:
                out = out + residual

        # Multi-head attention
        attn_out, _ = self.multihead_attention(out, out, out)
        out = out + attn_out  # Residual connection

        # Global average pooling
        out = torch.mean(out, dim=1)

        # Output layers
        out = self.fc1(out)
        out = self.relu(out)
        out = self.dropout(out)

        out = self.fc2(out)
        out = self.relu(out)
        out = self.dropout(out)

        predictions = self.fc3(out)

        return predictions


class EnsembleCombiner(nn.Module):
    """Combines LSTM price prediction with sentiment analysis."""

    def __init__(
        self,
        hidden_size: int = 64,
        dropout: float = 0.2
    ):
        """
        Initialize ensemble combiner.

        Args:
            hidden_size: Hidden layer size
            dropout: Dropout rate
        """
        super(EnsembleCombiner, self).__init__()

        # Combine price prediction and sentiment
        self.fc1 = nn.Linear(2, hidden_size)  # price_pred + sentiment
        self.fc2 = nn.Linear(hidden_size, hidden_size // 2)
        self.fc3 = nn.Linear(hidden_size // 2, 1)

        # Confidence estimation
        self.confidence_fc = nn.Linear(3, 1)  # price_pred + sentiment + sentiment_conf

        self.relu = nn.ReLU()
        self.dropout = nn.Dropout(dropout)
        self.sigmoid = nn.Sigmoid()

    def forward(
        self,
        price_pred: torch.Tensor,
        sentiment: torch.Tensor,
        sentiment_conf: torch.Tensor
    ) -> Tuple[torch.Tensor, torch.Tensor, torch.Tensor, torch.Tensor]:
        """
        Combine predictions.

        Args:
            price_pred: Price prediction (batch, 1)
            sentiment: Sentiment score (batch, 1)
            sentiment_conf: Sentiment confidence (batch, 1)

        Returns:
            Tuple of (final_pred, price_comp, sentiment_comp, confidence)
        """
        # Combine inputs
        combined = torch.cat([price_pred, sentiment], dim=1)

        # Forward pass
        out = self.fc1(combined)
        out = self.relu(out)
        out = self.dropout(out)

        out = self.fc2(out)
        out = self.relu(out)
        out = self.dropout(out)

        final_pred = self.fc3(out)

        # Calculate confidence
        conf_input = torch.cat([price_pred, sentiment, sentiment_conf], dim=1)
        confidence = self.sigmoid(self.confidence_fc(conf_input))

        return final_pred, price_pred, sentiment, confidence


def create_lstm_model(config: dict) -> nn.Module:
    """
    Factory function to create LSTM model.

    Args:
        config: Model configuration

    Returns:
        LSTM model instance
    """
    model_type = config.get('type', 'basic')

    if model_type == 'basic':
        return LSTMPredictor(
            input_size=config.get('input_size', 5),
            hidden_size=config.get('hidden_size', 128),
            num_layers=config.get('num_layers', 2),
            dropout=config.get('dropout', 0.2),
            bidirectional=config.get('bidirectional', False)
        )
    elif model_type == 'advanced':
        return AdvancedLSTMPredictor(
            input_size=config.get('input_size', 5),
            hidden_size=config.get('hidden_size', 256),
            num_layers=config.get('num_layers', 3),
            dropout=config.get('dropout', 0.3)
        )
    else:
        raise ValueError(f"Unknown model type: {model_type}")


def save_model_for_triton(
    model: nn.Module,
    save_path: str,
    example_input: Optional[torch.Tensor] = None
):
    """
    Save model in TorchScript format for Triton.

    Args:
        model: PyTorch model
        save_path: Path to save model
        example_input: Example input for tracing
    """
    try:
        model.eval()

        # Use scripting (preferred) or tracing
        if example_input is not None:
            # Tracing
            traced_model = torch.jit.trace(model, example_input)
        else:
            # Scripting
            traced_model = torch.jit.script(model)

        # Save
        traced_model.save(save_path)
        logger.info(f"Model saved to {save_path}")

    except Exception as e:
        logger.error(f"Failed to save model: {e}")
        raise
