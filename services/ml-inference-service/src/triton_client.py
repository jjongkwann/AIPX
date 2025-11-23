"""Triton Inference Server client wrapper."""

import logging
from typing import Any, Dict, List, Optional

import numpy as np
import tritonclient.grpc as grpcclient
import tritonclient.grpc.aio as aiogrpcclient
from tritonclient.utils import InferenceServerException

from .config import settings

logger = logging.getLogger(__name__)


class TritonClient:
    """Async Triton Inference Server client."""

    def __init__(self, url: Optional[str] = None):
        """
        Initialize Triton client.

        Args:
            url: Triton server URL (default: from settings)
        """
        self.url = url or settings.triton_url
        self.client: Optional[aiogrpcclient.InferenceServerClient] = None
        self._connected = False

    async def connect(self):
        """Establish connection to Triton server."""
        try:
            self.client = aiogrpcclient.InferenceServerClient(url=self.url, verbose=False)

            # Check server health
            await self.client.is_server_live()
            await self.client.is_server_ready()

            self._connected = True
            logger.info(f"Connected to Triton server at {self.url}")

            # Log available models
            models = await self.client.get_model_repository_index()
            logger.info(f"Available models: {[m.name for m in models]}")

        except Exception as e:
            logger.error(f"Failed to connect to Triton server: {e}")
            raise

    async def disconnect(self):
        """Close connection to Triton server."""
        if self.client:
            await self.client.close()
            self._connected = False
            logger.info("Disconnected from Triton server")

    async def is_ready(self) -> bool:
        """Check if Triton server is ready."""
        try:
            if not self.client:
                return False
            return await self.client.is_server_ready()
        except Exception:
            return False

    async def predict_price(self, features: np.ndarray, model_version: Optional[str] = None) -> Dict[str, Any]:
        """
        Predict next tick price using LSTM model.

        Args:
            features: shape (60, 5) - OHLCV features
            model_version: Model version to use (default: latest)

        Returns:
            Dictionary with prediction and metadata
        """
        try:
            if not self._connected:
                await self.connect()

            # Validate input shape
            if features.shape != (60, 5):
                raise ValueError(f"Expected features shape (60, 5), got {features.shape}")

            # Prepare input tensor
            features = features.astype(np.float32)
            input_tensor = grpcclient.InferInput("input__0", features.shape, "FP32")
            input_tensor.set_data_from_numpy(features)

            # Prepare output
            output_tensor = grpcclient.InferRequestedOutput("output__0")

            # Execute inference
            model_name = "lstm_price_predictor"
            response = await self.client.infer(
                model_name=model_name, inputs=[input_tensor], outputs=[output_tensor], model_version=model_version or ""
            )

            # Extract prediction
            prediction = response.as_numpy("output__0")[0]

            # Get model metadata
            stats = await self.client.get_inference_statistics(model_name)

            return {
                "predicted_price": float(prediction),
                "model_name": model_name,
                "model_version": model_version or "latest",
                "inference_time_ms": stats.model_stats[0].inference_stats.success.ns / 1e6
                if stats.model_stats
                else None,
            }

        except InferenceServerException as e:
            logger.error(f"Triton inference error: {e}")
            raise
        except Exception as e:
            logger.error(f"Price prediction error: {e}")
            raise

    async def analyze_sentiment(
        self, input_ids: np.ndarray, attention_mask: np.ndarray, model_version: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Analyze news sentiment using transformer model.

        Args:
            input_ids: Tokenized input IDs, shape (512,)
            attention_mask: Attention mask, shape (512,)
            model_version: Model version to use

        Returns:
            Dictionary with sentiment score and confidence
        """
        try:
            if not self._connected:
                await self.connect()

            # Validate input shapes
            if input_ids.shape != (512,):
                raise ValueError(f"Expected input_ids shape (512,), got {input_ids.shape}")
            if attention_mask.shape != (512,):
                raise ValueError(f"Expected attention_mask shape (512,), got {attention_mask.shape}")

            # Prepare inputs
            input_ids = input_ids.astype(np.int64)
            attention_mask = attention_mask.astype(np.int64)

            input_ids_tensor = grpcclient.InferInput("input_ids", input_ids.shape, "INT64")
            input_ids_tensor.set_data_from_numpy(input_ids)

            attention_mask_tensor = grpcclient.InferInput("attention_mask", attention_mask.shape, "INT64")
            attention_mask_tensor.set_data_from_numpy(attention_mask)

            # Prepare outputs
            sentiment_output = grpcclient.InferRequestedOutput("sentiment")
            confidence_output = grpcclient.InferRequestedOutput("confidence")

            # Execute inference
            model_name = "transformer_sentiment"
            response = await self.client.infer(
                model_name=model_name,
                inputs=[input_ids_tensor, attention_mask_tensor],
                outputs=[sentiment_output, confidence_output],
                model_version=model_version or "",
            )

            # Extract results
            sentiment = response.as_numpy("sentiment")[0]
            confidence = response.as_numpy("confidence")[0]

            return {
                "sentiment": float(sentiment),
                "confidence": float(confidence),
                "model_name": model_name,
                "model_version": model_version or "latest",
            }

        except InferenceServerException as e:
            logger.error(f"Triton inference error: {e}")
            raise
        except Exception as e:
            logger.error(f"Sentiment analysis error: {e}")
            raise

    async def ensemble_predict(
        self,
        price_features: np.ndarray,
        news_text_ids: np.ndarray,
        attention_mask: np.ndarray,
        model_version: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        Combined prediction using ensemble model.

        Args:
            price_features: OHLCV features, shape (60, 5)
            news_text_ids: Tokenized news, shape (512,)
            attention_mask: Attention mask, shape (512,)
            model_version: Model version to use

        Returns:
            Dictionary with ensemble prediction and components
        """
        try:
            if not self._connected:
                await self.connect()

            # Prepare inputs
            price_features = price_features.astype(np.float32)
            news_text_ids = news_text_ids.astype(np.int64)
            attention_mask = attention_mask.astype(np.int64)

            price_input = grpcclient.InferInput("price_features", price_features.shape, "FP32")
            price_input.set_data_from_numpy(price_features)

            text_input = grpcclient.InferInput("news_text_ids", news_text_ids.shape, "INT64")
            text_input.set_data_from_numpy(news_text_ids)

            mask_input = grpcclient.InferInput("attention_mask", attention_mask.shape, "INT64")
            mask_input.set_data_from_numpy(attention_mask)

            # Prepare outputs
            final_pred = grpcclient.InferRequestedOutput("final_prediction")
            price_comp = grpcclient.InferRequestedOutput("price_component")
            sentiment_comp = grpcclient.InferRequestedOutput("sentiment_component")
            confidence = grpcclient.InferRequestedOutput("confidence")

            # Execute inference
            model_name = "ensemble_predictor"
            response = await self.client.infer(
                model_name=model_name,
                inputs=[price_input, text_input, mask_input],
                outputs=[final_pred, price_comp, sentiment_comp, confidence],
                model_version=model_version or "",
            )

            return {
                "final_prediction": float(response.as_numpy("final_prediction")[0]),
                "price_component": float(response.as_numpy("price_component")[0]),
                "sentiment_component": float(response.as_numpy("sentiment_component")[0]),
                "confidence": float(response.as_numpy("confidence")[0]),
                "model_name": model_name,
                "model_version": model_version or "latest",
            }

        except InferenceServerException as e:
            logger.error(f"Triton inference error: {e}")
            raise
        except Exception as e:
            logger.error(f"Ensemble prediction error: {e}")
            raise

    async def batch_predict_prices(
        self, features_batch: List[np.ndarray], model_version: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """
        Batch price prediction for multiple samples.

        Args:
            features_batch: List of feature arrays
            model_version: Model version to use

        Returns:
            List of predictions
        """
        try:
            if not self._connected:
                await self.connect()

            # Stack features into batch
            batch = np.stack(features_batch, axis=0).astype(np.float32)

            # Prepare input
            input_tensor = grpcclient.InferInput("input__0", batch.shape, "FP32")
            input_tensor.set_data_from_numpy(batch)

            # Prepare output
            output_tensor = grpcclient.InferRequestedOutput("output__0")

            # Execute inference
            response = await self.client.infer(
                model_name="lstm_price_predictor",
                inputs=[input_tensor],
                outputs=[output_tensor],
                model_version=model_version or "",
            )

            # Extract predictions
            predictions = response.as_numpy("output__0")

            return [
                {"predicted_price": float(pred[0]), "batch_index": i, "model_version": model_version or "latest"}
                for i, pred in enumerate(predictions)
            ]

        except Exception as e:
            logger.error(f"Batch prediction error: {e}")
            raise

    async def get_model_metadata(self, model_name: str) -> Dict[str, Any]:
        """Get metadata for a specific model."""
        try:
            if not self._connected:
                await self.connect()

            metadata = await self.client.get_model_metadata(model_name)
            config = await self.client.get_model_config(model_name)

            return {
                "name": metadata.name,
                "versions": metadata.versions,
                "platform": metadata.platform,
                "inputs": [
                    {"name": inp.name, "datatype": inp.datatype, "shape": list(inp.shape)} for inp in metadata.inputs
                ],
                "outputs": [
                    {"name": out.name, "datatype": out.datatype, "shape": list(out.shape)} for out in metadata.outputs
                ],
                "max_batch_size": config.config.max_batch_size,
            }

        except Exception as e:
            logger.error(f"Failed to get model metadata: {e}")
            raise

    async def get_server_metrics(self) -> Dict[str, Any]:
        """Get Triton server metrics."""
        try:
            if not self._connected:
                await self.connect()

            # Get model statistics
            stats = await self.client.get_inference_statistics()

            metrics = {}
            for model_stat in stats.model_stats:
                model_name = model_stat.name
                success_stats = model_stat.inference_stats.success

                metrics[model_name] = {
                    "count": success_stats.count,
                    "total_time_ns": success_stats.ns,
                    "avg_time_ms": success_stats.ns / success_stats.count / 1e6 if success_stats.count > 0 else 0,
                    "version": model_stat.version,
                }

            return metrics

        except Exception as e:
            logger.error(f"Failed to get server metrics: {e}")
            raise


# Global client instance
triton_client = TritonClient()
