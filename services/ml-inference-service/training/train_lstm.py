"""Training pipeline for LSTM price prediction model."""

import asyncio
import json
import logging
import sys
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import numpy as np
import torch
import torch.nn as nn
import torch.optim as optim
from torch.utils.data import DataLoader, Dataset

sys.path.append(str(Path(__file__).parent.parent))

from src.config import settings
from src.models.lstm_model import AdvancedLSTMPredictor, LSTMPredictor, save_model_for_triton

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class PriceDataset(Dataset):
    """Dataset for price prediction."""

    def __init__(self, features: np.ndarray, targets: np.ndarray):
        """
        Initialize dataset.

        Args:
            features: Input features (N, 60, 5)
            targets: Target prices (N, 1)
        """
        self.features = torch.FloatTensor(features)
        self.targets = torch.FloatTensor(targets)

    def __len__(self) -> int:
        return len(self.features)

    def __getitem__(self, idx: int) -> Tuple[torch.Tensor, torch.Tensor]:
        return self.features[idx], self.targets[idx]


class ModelTrainer:
    """LSTM model trainer."""

    def __init__(
        self,
        model: nn.Module,
        device: str = "cuda" if torch.cuda.is_available() else "cpu",
        learning_rate: float = 0.001,
        weight_decay: float = 1e-5,
    ):
        """
        Initialize trainer.

        Args:
            model: Model to train
            device: Device to use
            learning_rate: Learning rate
            weight_decay: L2 regularization
        """
        self.model = model.to(device)
        self.device = device

        # Optimizer
        self.optimizer = optim.Adam(model.parameters(), lr=learning_rate, weight_decay=weight_decay)

        # Loss function
        self.criterion = nn.MSELoss()

        # Learning rate scheduler
        self.scheduler = optim.lr_scheduler.ReduceLROnPlateau(
            self.optimizer, mode="min", factor=0.5, patience=5, verbose=True
        )

        # Training history
        self.history = {"train_loss": [], "val_loss": [], "learning_rate": []}

    def train_epoch(self, train_loader: DataLoader, epoch: int) -> float:
        """
        Train for one epoch.

        Args:
            train_loader: Training data loader
            epoch: Current epoch

        Returns:
            Average training loss
        """
        self.model.train()
        total_loss = 0.0
        num_batches = 0

        for batch_idx, (features, targets) in enumerate(train_loader):
            # Move to device
            features = features.to(self.device)
            targets = targets.to(self.device)

            # Forward pass
            self.optimizer.zero_grad()
            predictions = self.model(features)
            loss = self.criterion(predictions, targets)

            # Backward pass
            loss.backward()

            # Gradient clipping
            torch.nn.utils.clip_grad_norm_(self.model.parameters(), max_norm=1.0)

            self.optimizer.step()

            total_loss += loss.item()
            num_batches += 1

            # Log progress
            if batch_idx % 100 == 0:
                logger.info(f"Epoch {epoch}, Batch {batch_idx}/{len(train_loader)}, Loss: {loss.item():.6f}")

        avg_loss = total_loss / num_batches
        return avg_loss

    def validate(self, val_loader: DataLoader) -> Dict[str, float]:
        """
        Validate model.

        Args:
            val_loader: Validation data loader

        Returns:
            Dictionary of validation metrics
        """
        self.model.eval()
        total_loss = 0.0
        all_predictions = []
        all_targets = []

        with torch.no_grad():
            for features, targets in val_loader:
                features = features.to(self.device)
                targets = targets.to(self.device)

                predictions = self.model(features)
                loss = self.criterion(predictions, targets)

                total_loss += loss.item()
                all_predictions.append(predictions.cpu().numpy())
                all_targets.append(targets.cpu().numpy())

        # Calculate metrics
        avg_loss = total_loss / len(val_loader)
        all_predictions = np.concatenate(all_predictions)
        all_targets = np.concatenate(all_targets)

        mae = np.mean(np.abs(all_predictions - all_targets))
        rmse = np.sqrt(np.mean((all_predictions - all_targets) ** 2))

        # Directional accuracy
        pred_direction = np.diff(all_predictions.flatten()) > 0
        true_direction = np.diff(all_targets.flatten()) > 0
        directional_accuracy = np.mean(pred_direction == true_direction) if len(pred_direction) > 0 else 0.0

        return {"loss": avg_loss, "mae": mae, "rmse": rmse, "directional_accuracy": directional_accuracy}

    def train(
        self,
        train_loader: DataLoader,
        val_loader: DataLoader,
        num_epochs: int,
        checkpoint_dir: Optional[Path] = None,
        early_stopping_patience: int = 10,
    ) -> Dict[str, List[float]]:
        """
        Train model.

        Args:
            train_loader: Training data loader
            val_loader: Validation data loader
            num_epochs: Number of epochs
            checkpoint_dir: Directory to save checkpoints
            early_stopping_patience: Patience for early stopping

        Returns:
            Training history
        """
        best_val_loss = float("inf")
        patience_counter = 0

        for epoch in range(1, num_epochs + 1):
            logger.info(f"\nEpoch {epoch}/{num_epochs}")
            logger.info("-" * 50)

            # Train
            train_loss = self.train_epoch(train_loader, epoch)
            logger.info(f"Training Loss: {train_loss:.6f}")

            # Validate
            val_metrics = self.validate(val_loader)
            val_loss = val_metrics["loss"]
            logger.info(f"Validation Loss: {val_loss:.6f}")
            logger.info(f"Validation MAE: {val_metrics['mae']:.6f}")
            logger.info(f"Validation RMSE: {val_metrics['rmse']:.6f}")
            logger.info(f"Directional Accuracy: {val_metrics['directional_accuracy']:.4f}")

            # Update learning rate
            self.scheduler.step(val_loss)
            current_lr = self.optimizer.param_groups[0]["lr"]

            # Update history
            self.history["train_loss"].append(train_loss)
            self.history["val_loss"].append(val_loss)
            self.history["learning_rate"].append(current_lr)

            # Save best model
            if val_loss < best_val_loss:
                best_val_loss = val_loss
                patience_counter = 0

                if checkpoint_dir:
                    checkpoint_path = checkpoint_dir / "best_model.pt"
                    torch.save(
                        {
                            "epoch": epoch,
                            "model_state_dict": self.model.state_dict(),
                            "optimizer_state_dict": self.optimizer.state_dict(),
                            "val_loss": val_loss,
                            "val_metrics": val_metrics,
                        },
                        checkpoint_path,
                    )
                    logger.info(f"Saved best model to {checkpoint_path}")
            else:
                patience_counter += 1

            # Early stopping
            if patience_counter >= early_stopping_patience:
                logger.info(f"Early stopping triggered after {epoch} epochs")
                break

        return self.history


async def load_training_data(
    start_date: str, end_date: str, symbols: List[str], window_size: int = 60
) -> Tuple[np.ndarray, np.ndarray]:
    """
    Load training data from database.

    Args:
        start_date: Start date
        end_date: End date
        symbols: List of symbols
        window_size: Feature window size

    Returns:
        Tuple of (features, targets)
    """
    # TODO: Replace with actual database query
    logger.info(f"Loading data for {symbols} from {start_date} to {end_date}")

    # For now, generate dummy data
    num_samples = 10000
    features = np.random.randn(num_samples, window_size, 5).astype(np.float32)
    targets = np.random.randn(num_samples, 1).astype(np.float32)

    return features, targets


def prepare_data(
    features: np.ndarray, targets: np.ndarray, train_split: float = 0.8, batch_size: int = 64
) -> Tuple[DataLoader, DataLoader]:
    """
    Prepare data loaders.

    Args:
        features: Input features
        targets: Target values
        train_split: Train/val split ratio
        batch_size: Batch size

    Returns:
        Tuple of (train_loader, val_loader)
    """
    # Split data
    split_idx = int(len(features) * train_split)

    train_features = features[:split_idx]
    train_targets = targets[:split_idx]
    val_features = features[split_idx:]
    val_targets = targets[split_idx:]

    # Create datasets
    train_dataset = PriceDataset(train_features, train_targets)
    val_dataset = PriceDataset(val_features, val_targets)

    # Create data loaders
    train_loader = DataLoader(train_dataset, batch_size=batch_size, shuffle=True, num_workers=4, pin_memory=True)

    val_loader = DataLoader(val_dataset, batch_size=batch_size, shuffle=False, num_workers=4, pin_memory=True)

    return train_loader, val_loader


async def main():
    """Main training pipeline."""
    logger.info("Starting LSTM training pipeline")

    # Configuration
    config = {
        "type": "basic",  # or 'advanced'
        "input_size": 5,
        "hidden_size": 128,
        "num_layers": 2,
        "dropout": 0.2,
        "learning_rate": 0.001,
        "batch_size": 64,
        "num_epochs": 100,
        "early_stopping_patience": 10,
    }

    # Create checkpoint directory
    checkpoint_dir = Path(settings.model_checkpoint_path)
    checkpoint_dir.mkdir(parents=True, exist_ok=True)

    # Load data
    logger.info("Loading training data...")
    features, targets = await load_training_data(
        start_date="2023-01-01", end_date="2023-12-31", symbols=["005930", "000660"]
    )

    # Prepare data loaders
    logger.info("Preparing data loaders...")
    train_loader, val_loader = prepare_data(features, targets, batch_size=config["batch_size"])

    # Create model
    logger.info("Creating model...")
    if config["type"] == "basic":
        model = LSTMPredictor(
            input_size=config["input_size"],
            hidden_size=config["hidden_size"],
            num_layers=config["num_layers"],
            dropout=config["dropout"],
        )
    else:
        model = AdvancedLSTMPredictor(
            input_size=config["input_size"],
            hidden_size=config["hidden_size"],
            num_layers=config["num_layers"],
            dropout=config["dropout"],
        )

    logger.info(f"Model parameters: {sum(p.numel() for p in model.parameters()):,}")

    # Create trainer
    trainer = ModelTrainer(model=model, learning_rate=config["learning_rate"])

    # Train
    logger.info("Starting training...")
    history = trainer.train(
        train_loader=train_loader,
        val_loader=val_loader,
        num_epochs=config["num_epochs"],
        checkpoint_dir=checkpoint_dir,
        early_stopping_patience=config["early_stopping_patience"],
    )

    # Save history
    history_path = checkpoint_dir / "training_history.json"
    with open(history_path, "w") as f:
        json.dump(history, f, indent=2)

    # Save model for Triton
    logger.info("Saving model for Triton...")
    triton_model_path = Path(settings.model_repository) / "lstm_price_predictor" / "1" / "model.pt"
    triton_model_path.parent.mkdir(parents=True, exist_ok=True)

    # Create example input for tracing
    example_input = torch.randn(1, 60, 5)

    save_model_for_triton(model=trainer.model, save_path=str(triton_model_path), example_input=example_input)

    logger.info("Training completed!")


if __name__ == "__main__":
    asyncio.run(main())
