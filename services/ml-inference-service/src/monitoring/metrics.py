"""Metrics collection and monitoring."""
from collections import defaultdict
from datetime import datetime, timedelta
from typing import Dict, Any, List
import asyncio
import logging
from dataclasses import dataclass, field

logger = logging.getLogger(__name__)


@dataclass
class InferenceMetrics:
    """Inference metrics for a model."""
    total_requests: int = 0
    successful_requests: int = 0
    failed_requests: int = 0
    total_inference_time_ms: float = 0.0
    min_inference_time_ms: float = float('inf')
    max_inference_time_ms: float = 0.0
    inference_times: List[float] = field(default_factory=list)

    @property
    def avg_inference_time_ms(self) -> float:
        """Calculate average inference time."""
        if self.successful_requests == 0:
            return 0.0
        return self.total_inference_time_ms / self.successful_requests

    @property
    def success_rate(self) -> float:
        """Calculate success rate."""
        if self.total_requests == 0:
            return 0.0
        return self.successful_requests / self.total_requests

    @property
    def p50_inference_time_ms(self) -> float:
        """Calculate p50 inference time."""
        if not self.inference_times:
            return 0.0
        sorted_times = sorted(self.inference_times)
        idx = len(sorted_times) // 2
        return sorted_times[idx]

    @property
    def p95_inference_time_ms(self) -> float:
        """Calculate p95 inference time."""
        if not self.inference_times:
            return 0.0
        sorted_times = sorted(self.inference_times)
        idx = int(len(sorted_times) * 0.95)
        return sorted_times[min(idx, len(sorted_times) - 1)]

    @property
    def p99_inference_time_ms(self) -> float:
        """Calculate p99 inference time."""
        if not self.inference_times:
            return 0.0
        sorted_times = sorted(self.inference_times)
        idx = int(len(sorted_times) * 0.99)
        return sorted_times[min(idx, len(sorted_times) - 1)]


class MetricsCollector:
    """Collect and aggregate inference metrics."""

    def __init__(self, max_inference_times: int = 1000):
        """
        Initialize metrics collector.

        Args:
            max_inference_times: Maximum number of inference times to keep
        """
        self.max_inference_times = max_inference_times
        self.metrics: Dict[str, InferenceMetrics] = defaultdict(InferenceMetrics)
        self.start_time = datetime.utcnow()
        self._lock = asyncio.Lock()

    async def record_inference(
        self,
        model_name: str,
        inference_time_ms: float,
        success: bool
    ):
        """
        Record inference metrics.

        Args:
            model_name: Name of the model
            inference_time_ms: Inference time in milliseconds
            success: Whether inference was successful
        """
        async with self._lock:
            metrics = self.metrics[model_name]

            metrics.total_requests += 1

            if success:
                metrics.successful_requests += 1
                metrics.total_inference_time_ms += inference_time_ms
                metrics.min_inference_time_ms = min(
                    metrics.min_inference_time_ms,
                    inference_time_ms
                )
                metrics.max_inference_time_ms = max(
                    metrics.max_inference_time_ms,
                    inference_time_ms
                )

                # Keep limited history for percentile calculations
                metrics.inference_times.append(inference_time_ms)
                if len(metrics.inference_times) > self.max_inference_times:
                    metrics.inference_times.pop(0)
            else:
                metrics.failed_requests += 1

    def get_metrics(self, model_name: Optional[str] = None) -> Dict[str, Any]:
        """
        Get collected metrics.

        Args:
            model_name: Specific model name (None for all models)

        Returns:
            Dictionary of metrics
        """
        uptime = (datetime.utcnow() - self.start_time).total_seconds()

        if model_name:
            if model_name not in self.metrics:
                return {}

            metrics = self.metrics[model_name]
            return {
                "model_name": model_name,
                "total_requests": metrics.total_requests,
                "successful_requests": metrics.successful_requests,
                "failed_requests": metrics.failed_requests,
                "success_rate": metrics.success_rate,
                "avg_inference_time_ms": metrics.avg_inference_time_ms,
                "min_inference_time_ms": metrics.min_inference_time_ms
                if metrics.min_inference_time_ms != float('inf') else 0.0,
                "max_inference_time_ms": metrics.max_inference_time_ms,
                "p50_inference_time_ms": metrics.p50_inference_time_ms,
                "p95_inference_time_ms": metrics.p95_inference_time_ms,
                "p99_inference_time_ms": metrics.p99_inference_time_ms,
                "requests_per_second": metrics.total_requests / uptime if uptime > 0 else 0.0
            }

        # Return all metrics
        all_metrics = {}
        for model, metrics in self.metrics.items():
            all_metrics[model] = {
                "total_requests": metrics.total_requests,
                "successful_requests": metrics.successful_requests,
                "failed_requests": metrics.failed_requests,
                "success_rate": metrics.success_rate,
                "avg_inference_time_ms": metrics.avg_inference_time_ms,
                "min_inference_time_ms": metrics.min_inference_time_ms
                if metrics.min_inference_time_ms != float('inf') else 0.0,
                "max_inference_time_ms": metrics.max_inference_time_ms,
                "p50_inference_time_ms": metrics.p50_inference_time_ms,
                "p95_inference_time_ms": metrics.p95_inference_time_ms,
                "p99_inference_time_ms": metrics.p99_inference_time_ms,
                "requests_per_second": metrics.total_requests / uptime if uptime > 0 else 0.0
            }

        return {
            "models": all_metrics,
            "uptime_seconds": uptime,
            "start_time": self.start_time.isoformat()
        }

    def reset_metrics(self, model_name: Optional[str] = None):
        """
        Reset metrics.

        Args:
            model_name: Specific model to reset (None for all)
        """
        if model_name:
            if model_name in self.metrics:
                self.metrics[model_name] = InferenceMetrics()
        else:
            self.metrics.clear()
            self.start_time = datetime.utcnow()


class ModelMonitor:
    """Monitor model performance and detect drift."""

    def __init__(
        self,
        alert_threshold_ms: float = 1000.0,
        error_rate_threshold: float = 0.05
    ):
        """
        Initialize model monitor.

        Args:
            alert_threshold_ms: Alert if inference time exceeds this
            error_rate_threshold: Alert if error rate exceeds this
        """
        self.alert_threshold_ms = alert_threshold_ms
        self.error_rate_threshold = error_rate_threshold
        self.alerts: List[Dict[str, Any]] = []

    def check_health(self, metrics: Dict[str, Any]) -> List[Dict[str, Any]]:
        """
        Check model health and generate alerts.

        Args:
            metrics: Metrics dictionary

        Returns:
            List of alerts
        """
        alerts = []

        for model_name, model_metrics in metrics.get("models", {}).items():
            # Check inference time
            if model_metrics["avg_inference_time_ms"] > self.alert_threshold_ms:
                alerts.append({
                    "severity": "warning",
                    "model": model_name,
                    "type": "high_latency",
                    "message": f"Average inference time ({model_metrics['avg_inference_time_ms']:.2f}ms) "
                               f"exceeds threshold ({self.alert_threshold_ms}ms)",
                    "timestamp": datetime.utcnow().isoformat()
                })

            # Check error rate
            error_rate = 1.0 - model_metrics["success_rate"]
            if error_rate > self.error_rate_threshold:
                alerts.append({
                    "severity": "critical",
                    "model": model_name,
                    "type": "high_error_rate",
                    "message": f"Error rate ({error_rate:.2%}) exceeds threshold ({self.error_rate_threshold:.2%})",
                    "timestamp": datetime.utcnow().isoformat()
                })

            # Check p99 latency
            if model_metrics["p99_inference_time_ms"] > self.alert_threshold_ms * 2:
                alerts.append({
                    "severity": "warning",
                    "model": model_name,
                    "type": "high_p99_latency",
                    "message": f"P99 latency ({model_metrics['p99_inference_time_ms']:.2f}ms) is very high",
                    "timestamp": datetime.utcnow().isoformat()
                })

        self.alerts.extend(alerts)
        return alerts

    def get_recent_alerts(self, hours: int = 24) -> List[Dict[str, Any]]:
        """
        Get recent alerts.

        Args:
            hours: Number of hours to look back

        Returns:
            List of recent alerts
        """
        cutoff = datetime.utcnow() - timedelta(hours=hours)

        recent_alerts = [
            alert for alert in self.alerts
            if datetime.fromisoformat(alert["timestamp"]) > cutoff
        ]

        return recent_alerts


# Global instances
metrics_collector = MetricsCollector()
model_monitor = ModelMonitor()
