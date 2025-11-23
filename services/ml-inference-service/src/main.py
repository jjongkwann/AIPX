"""FastAPI gateway for ML Inference Service."""

import logging
from contextlib import asynccontextmanager
from datetime import datetime
from typing import Any, Dict, List, Optional

import numpy as np
from fastapi import BackgroundTasks, FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

from .config import settings
from .features.feature_extractor import sentiment_feature_extractor
from .monitoring.metrics import MetricsCollector
from .triton_client import triton_client

# Setup logging
logging.basicConfig(level=settings.log_level, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

# Metrics collector
metrics_collector = MetricsCollector()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifespan context manager for startup/shutdown."""
    # Startup
    logger.info("Starting ML Inference Service...")
    try:
        await triton_client.connect()
        logger.info("Connected to Triton server")
    except Exception as e:
        logger.error(f"Failed to connect to Triton: {e}")

    yield

    # Shutdown
    logger.info("Shutting down ML Inference Service...")
    await triton_client.disconnect()


app = FastAPI(
    title="AIPX ML Inference Service",
    description="Machine Learning inference service using NVIDIA Triton",
    version="1.0.0",
    lifespan=lifespan,
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# Request/Response Models
class PredictPriceRequest(BaseModel):
    """Price prediction request."""

    symbol: str = Field(..., description="Stock symbol")
    ohlcv_data: List[List[float]] = Field(..., description="OHLCV data, shape (60, 5)")
    model_version: Optional[str] = Field(None, description="Model version")


class PredictPriceResponse(BaseModel):
    """Price prediction response."""

    predicted_price: float
    confidence: float
    model_name: str
    model_version: str
    inference_time_ms: Optional[float] = None
    timestamp: datetime


class SentimentRequest(BaseModel):
    """Sentiment analysis request."""

    text: str = Field(..., description="Text to analyze")
    model_version: Optional[str] = Field(None, description="Model version")


class SentimentResponse(BaseModel):
    """Sentiment analysis response."""

    sentiment: float = Field(..., description="Sentiment score (-1 to +1)")
    confidence: float
    model_name: str
    model_version: str
    timestamp: datetime


class EnsemblePredictRequest(BaseModel):
    """Ensemble prediction request."""

    symbol: str
    ohlcv_data: List[List[float]]
    news_text: str
    model_version: Optional[str] = None


class EnsemblePredictResponse(BaseModel):
    """Ensemble prediction response."""

    final_prediction: float
    price_component: float
    sentiment_component: float
    confidence: float
    model_name: str
    model_version: str
    timestamp: datetime


class BatchPredictRequest(BaseModel):
    """Batch prediction request."""

    items: List[Dict[str, Any]]
    model_version: Optional[str] = None


class HealthResponse(BaseModel):
    """Health check response."""

    status: str
    triton_connected: bool
    available_models: List[str]
    timestamp: datetime


# Health Check
@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    try:
        is_ready = await triton_client.is_ready()

        # Get available models
        available_models = []
        if is_ready:
            try:
                # This would query Triton for available models
                available_models = ["lstm_price_predictor", "transformer_sentiment", "ensemble_predictor"]
            except Exception as e:
                logger.warning(f"Failed to get available models: {e}")

        return HealthResponse(
            status="healthy" if is_ready else "degraded",
            triton_connected=is_ready,
            available_models=available_models,
            timestamp=datetime.utcnow(),
        )

    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return JSONResponse(
            status_code=503,
            content={
                "status": "unhealthy",
                "triton_connected": False,
                "available_models": [],
                "timestamp": datetime.utcnow().isoformat(),
                "error": str(e),
            },
        )


# Price Prediction
@app.post("/api/v1/predict/price", response_model=PredictPriceResponse)
async def predict_price(request: PredictPriceRequest, background_tasks: BackgroundTasks):
    """
    Predict next tick price using LSTM model.

    Example request:
    ```json
    {
        "symbol": "005930",
        "ohlcv_data": [[...], [...], ...],  # 60 rows, 5 columns
        "model_version": "1"
    }
    ```
    """
    try:
        start_time = datetime.utcnow()

        # Validate input
        features = np.array(request.ohlcv_data, dtype=np.float32)
        if features.shape != (60, 5):
            raise HTTPException(status_code=400, detail=f"Expected ohlcv_data shape (60, 5), got {features.shape}")

        # Make prediction
        result = await triton_client.predict_price(features=features, model_version=request.model_version)

        # Calculate confidence (simple heuristic)
        confidence = min(0.95, max(0.5, 1.0 - np.std(features[:, 0]) / np.mean(features[:, 0])))

        # Record metrics
        inference_time = (datetime.utcnow() - start_time).total_seconds() * 1000
        background_tasks.add_task(
            metrics_collector.record_inference,
            model_name="lstm_price_predictor",
            inference_time_ms=inference_time,
            success=True,
        )

        return PredictPriceResponse(
            predicted_price=result["predicted_price"],
            confidence=confidence,
            model_name=result["model_name"],
            model_version=result["model_version"],
            inference_time_ms=inference_time,
            timestamp=datetime.utcnow(),
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Price prediction error: {e}")
        background_tasks.add_task(
            metrics_collector.record_inference, model_name="lstm_price_predictor", inference_time_ms=0, success=False
        )
        raise HTTPException(status_code=500, detail=str(e))


# Sentiment Analysis
@app.post("/api/v1/predict/sentiment", response_model=SentimentResponse)
async def predict_sentiment(request: SentimentRequest, background_tasks: BackgroundTasks):
    """
    Analyze sentiment of news text.

    Example request:
    ```json
    {
        "text": "Samsung reports strong quarterly earnings...",
        "model_version": "1"
    }
    ```
    """
    try:
        start_time = datetime.utcnow()

        # Tokenize text
        tokens = sentiment_feature_extractor.tokenize_text(request.text)

        # Make prediction
        result = await triton_client.analyze_sentiment(
            input_ids=tokens["input_ids"], attention_mask=tokens["attention_mask"], model_version=request.model_version
        )

        # Record metrics
        inference_time = (datetime.utcnow() - start_time).total_seconds() * 1000
        background_tasks.add_task(
            metrics_collector.record_inference,
            model_name="transformer_sentiment",
            inference_time_ms=inference_time,
            success=True,
        )

        return SentimentResponse(
            sentiment=result["sentiment"],
            confidence=result["confidence"],
            model_name=result["model_name"],
            model_version=result["model_version"],
            timestamp=datetime.utcnow(),
        )

    except Exception as e:
        logger.error(f"Sentiment prediction error: {e}")
        background_tasks.add_task(
            metrics_collector.record_inference, model_name="transformer_sentiment", inference_time_ms=0, success=False
        )
        raise HTTPException(status_code=500, detail=str(e))


# Ensemble Prediction
@app.post("/api/v1/predict/ensemble", response_model=EnsemblePredictResponse)
async def predict_ensemble(request: EnsemblePredictRequest, background_tasks: BackgroundTasks):
    """
    Combined prediction using both price and sentiment models.

    Example request:
    ```json
    {
        "symbol": "005930",
        "ohlcv_data": [[...], [...], ...],
        "news_text": "Samsung announces new AI chip...",
        "model_version": "1"
    }
    ```
    """
    try:
        start_time = datetime.utcnow()

        # Prepare price features
        price_features = np.array(request.ohlcv_data, dtype=np.float32)
        if price_features.shape != (60, 5):
            raise HTTPException(
                status_code=400, detail=f"Expected ohlcv_data shape (60, 5), got {price_features.shape}"
            )

        # Tokenize news text
        tokens = sentiment_feature_extractor.tokenize_text(request.news_text)

        # Make ensemble prediction
        result = await triton_client.ensemble_predict(
            price_features=price_features,
            news_text_ids=tokens["input_ids"],
            attention_mask=tokens["attention_mask"],
            model_version=request.model_version,
        )

        # Record metrics
        inference_time = (datetime.utcnow() - start_time).total_seconds() * 1000
        background_tasks.add_task(
            metrics_collector.record_inference,
            model_name="ensemble_predictor",
            inference_time_ms=inference_time,
            success=True,
        )

        return EnsemblePredictResponse(
            final_prediction=result["final_prediction"],
            price_component=result["price_component"],
            sentiment_component=result["sentiment_component"],
            confidence=result["confidence"],
            model_name=result["model_name"],
            model_version=result["model_version"],
            timestamp=datetime.utcnow(),
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Ensemble prediction error: {e}")
        background_tasks.add_task(
            metrics_collector.record_inference, model_name="ensemble_predictor", inference_time_ms=0, success=False
        )
        raise HTTPException(status_code=500, detail=str(e))


# Batch Prediction
@app.post("/api/v1/predict/batch")
async def predict_batch(request: BatchPredictRequest):
    """
    Batch prediction for multiple samples.

    Example request:
    ```json
    {
        "items": [
            {"symbol": "005930", "ohlcv_data": [...]},
            {"symbol": "000660", "ohlcv_data": [...]}
        ],
        "model_version": "1"
    }
    ```
    """
    try:
        features_batch = []
        for item in request.items:
            features = np.array(item["ohlcv_data"], dtype=np.float32)
            if features.shape != (60, 5):
                raise HTTPException(status_code=400, detail=f"Invalid shape for item: {features.shape}")
            features_batch.append(features)

        # Make batch prediction
        results = await triton_client.batch_predict_prices(
            features_batch=features_batch, model_version=request.model_version
        )

        # Add symbol information
        for i, item in enumerate(request.items):
            results[i]["symbol"] = item.get("symbol", "unknown")

        return {"predictions": results, "count": len(results), "timestamp": datetime.utcnow()}

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Batch prediction error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


# Model Metadata
@app.get("/api/v1/models/{model_name}/metadata")
async def get_model_metadata(model_name: str):
    """Get metadata for a specific model."""
    try:
        metadata = await triton_client.get_model_metadata(model_name)
        return metadata
    except Exception as e:
        logger.error(f"Failed to get model metadata: {e}")
        raise HTTPException(status_code=404, detail=f"Model {model_name} not found")


# Metrics
@app.get("/api/v1/metrics")
async def get_metrics():
    """Get inference metrics."""
    try:
        server_metrics = await triton_client.get_server_metrics()
        app_metrics = metrics_collector.get_metrics()

        return {"server_metrics": server_metrics, "app_metrics": app_metrics, "timestamp": datetime.utcnow()}
    except Exception as e:
        logger.error(f"Failed to get metrics: {e}")
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app", host="0.0.0.0", port=settings.service_port, reload=True, log_level=settings.log_level.lower()
    )
