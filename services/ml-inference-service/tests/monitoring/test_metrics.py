"""Tests for metrics collection."""
import pytest
import asyncio
from src.monitoring.metrics import MetricsCollector, ModelMonitor, InferenceMetrics


@pytest.mark.monitoring
class TestInferenceMetrics:
    """Test InferenceMetrics dataclass."""
    
    def test_metrics_initialization(self):
        """Test metrics initialization."""
        metrics = InferenceMetrics()
        
        assert metrics.total_requests == 0
        assert metrics.successful_requests == 0
        assert metrics.failed_requests == 0
        assert metrics.total_inference_time_ms == 0.0
        assert metrics.min_inference_time_ms == float('inf')
        assert metrics.max_inference_time_ms == 0.0
        assert len(metrics.inference_times) == 0
    
    def test_avg_inference_time(self):
        """Test average inference time calculation."""
        metrics = InferenceMetrics()
        metrics.successful_requests = 10
        metrics.total_inference_time_ms = 125.0
        
        assert metrics.avg_inference_time_ms == 12.5
    
    def test_avg_inference_time_zero_requests(self):
        """Test average with zero requests."""
        metrics = InferenceMetrics()
        assert metrics.avg_inference_time_ms == 0.0
    
    def test_success_rate(self):
        """Test success rate calculation."""
        metrics = InferenceMetrics()
        metrics.total_requests = 100
        metrics.successful_requests = 95
        
        assert metrics.success_rate == 0.95
    
    def test_success_rate_zero_requests(self):
        """Test success rate with zero requests."""
        metrics = InferenceMetrics()
        assert metrics.success_rate == 0.0
    
    def test_percentile_calculations(self):
        """Test percentile calculations."""
        metrics = InferenceMetrics()
        metrics.inference_times = list(range(1, 101))  # 1 to 100

        # np.percentile uses linear interpolation by default
        assert 49 <= metrics.p50_inference_time_ms <= 51
        assert 94 <= metrics.p95_inference_time_ms <= 96
        assert 98 <= metrics.p99_inference_time_ms <= 100
    
    def test_percentiles_empty_list(self):
        """Test percentiles with empty list."""
        metrics = InferenceMetrics()
        
        assert metrics.p50_inference_time_ms == 0.0
        assert metrics.p95_inference_time_ms == 0.0
        assert metrics.p99_inference_time_ms == 0.0


@pytest.mark.monitoring
@pytest.mark.asyncio
class TestMetricsCollector:
    """Test MetricsCollector class."""
    
    async def test_collector_initialization(self):
        """Test collector initialization."""
        collector = MetricsCollector(max_inference_times=100)
        
        assert collector.max_inference_times == 100
        assert len(collector.metrics) == 0
        assert collector.start_time is not None
    
    async def test_record_successful_inference(self):
        """Test recording successful inference."""
        collector = MetricsCollector()
        
        await collector.record_inference(
            model_name="test_model",
            inference_time_ms=12.5,
            success=True
        )
        
        metrics = collector.metrics["test_model"]
        assert metrics.total_requests == 1
        assert metrics.successful_requests == 1
        assert metrics.failed_requests == 0
        assert metrics.total_inference_time_ms == 12.5
        assert metrics.min_inference_time_ms == 12.5
        assert metrics.max_inference_time_ms == 12.5
        assert len(metrics.inference_times) == 1
    
    async def test_record_failed_inference(self):
        """Test recording failed inference."""
        collector = MetricsCollector()
        
        await collector.record_inference(
            model_name="test_model",
            inference_time_ms=0.0,
            success=False
        )
        
        metrics = collector.metrics["test_model"]
        assert metrics.total_requests == 1
        assert metrics.successful_requests == 0
        assert metrics.failed_requests == 1
    
    async def test_record_multiple_inferences(self):
        """Test recording multiple inferences."""
        collector = MetricsCollector()
        
        for i in range(10):
            await collector.record_inference(
                model_name="test_model",
                inference_time_ms=10.0 + i,
                success=True
            )
        
        metrics = collector.metrics["test_model"]
        assert metrics.total_requests == 10
        assert metrics.successful_requests == 10
        assert metrics.min_inference_time_ms == 10.0
        assert metrics.max_inference_time_ms == 19.0
        assert len(metrics.inference_times) == 10
    
    async def test_inference_times_limit(self):
        """Test inference times list is limited."""
        collector = MetricsCollector(max_inference_times=5)
        
        for i in range(10):
            await collector.record_inference(
                model_name="test_model",
                inference_time_ms=float(i),
                success=True
            )
        
        metrics = collector.metrics["test_model"]
        # Should only keep last 5
        assert len(metrics.inference_times) == 5
        # Should be last 5 values: [5, 6, 7, 8, 9]
        assert metrics.inference_times == [5.0, 6.0, 7.0, 8.0, 9.0]
    
    async def test_multiple_models(self):
        """Test tracking multiple models."""
        collector = MetricsCollector()
        
        await collector.record_inference("model_a", 10.0, True)
        await collector.record_inference("model_b", 20.0, True)
        await collector.record_inference("model_a", 15.0, True)
        
        assert "model_a" in collector.metrics
        assert "model_b" in collector.metrics
        assert collector.metrics["model_a"].total_requests == 2
        assert collector.metrics["model_b"].total_requests == 1
    
    def test_get_metrics_single_model(self):
        """Test getting metrics for single model."""
        collector = MetricsCollector()
        collector.metrics["test_model"] = InferenceMetrics(
            total_requests=100,
            successful_requests=95,
            failed_requests=5,
            total_inference_time_ms=1250.0,
            min_inference_time_ms=10.0,
            max_inference_time_ms=25.0,
            inference_times=[12.0] * 100
        )
        
        metrics = collector.get_metrics(model_name="test_model")
        
        assert metrics["model_name"] == "test_model"
        assert metrics["total_requests"] == 100
        assert metrics["successful_requests"] == 95
        assert metrics["failed_requests"] == 5
        assert metrics["success_rate"] == 0.95
        assert metrics["avg_inference_time_ms"] > 0
        assert "requests_per_second" in metrics
    
    def test_get_metrics_all_models(self):
        """Test getting metrics for all models."""
        collector = MetricsCollector()
        collector.metrics["model_a"] = InferenceMetrics(total_requests=10)
        collector.metrics["model_b"] = InferenceMetrics(total_requests=20)
        
        metrics = collector.get_metrics()
        
        assert "models" in metrics
        assert "uptime_seconds" in metrics
        assert "start_time" in metrics
        assert "model_a" in metrics["models"]
        assert "model_b" in metrics["models"]
    
    def test_get_metrics_nonexistent_model(self):
        """Test getting metrics for non-existent model."""
        collector = MetricsCollector()
        
        metrics = collector.get_metrics(model_name="nonexistent")
        
        assert metrics == {}
    
    def test_reset_metrics_single_model(self):
        """Test resetting metrics for single model."""
        collector = MetricsCollector()
        collector.metrics["test_model"] = InferenceMetrics(total_requests=100)
        
        collector.reset_metrics(model_name="test_model")
        
        assert collector.metrics["test_model"].total_requests == 0
    
    def test_reset_metrics_all(self):
        """Test resetting all metrics."""
        collector = MetricsCollector()
        collector.metrics["model_a"] = InferenceMetrics(total_requests=10)
        collector.metrics["model_b"] = InferenceMetrics(total_requests=20)
        
        start_time_before = collector.start_time
        collector.reset_metrics()
        
        assert len(collector.metrics) == 0
        assert collector.start_time > start_time_before
    
    async def test_concurrent_recording(self):
        """Test concurrent metric recording."""
        collector = MetricsCollector()
        
        tasks = [
            collector.record_inference("test_model", 10.0, True)
            for _ in range(100)
        ]
        
        await asyncio.gather(*tasks)
        
        metrics = collector.metrics["test_model"]
        assert metrics.total_requests == 100


@pytest.mark.monitoring
class TestModelMonitor:
    """Test ModelMonitor class."""
    
    def test_monitor_initialization(self):
        """Test monitor initialization."""
        monitor = ModelMonitor(
            alert_threshold_ms=1000.0,
            error_rate_threshold=0.05
        )
        
        assert monitor.alert_threshold_ms == 1000.0
        assert monitor.error_rate_threshold == 0.05
        assert len(monitor.alerts) == 0
    
    def test_check_health_no_issues(self):
        """Test health check with no issues."""
        monitor = ModelMonitor()
        
        metrics = {
            "models": {
                "test_model": {
                    "avg_inference_time_ms": 10.0,
                    "p99_inference_time_ms": 20.0,
                    "success_rate": 0.99
                }
            }
        }
        
        alerts = monitor.check_health(metrics)
        
        assert len(alerts) == 0
    
    def test_check_health_high_latency(self):
        """Test health check detects high latency."""
        monitor = ModelMonitor(alert_threshold_ms=10.0)
        
        metrics = {
            "models": {
                "test_model": {
                    "avg_inference_time_ms": 15.0,
                    "p99_inference_time_ms": 25.0,
                    "success_rate": 0.99
                }
            }
        }
        
        alerts = monitor.check_health(metrics)
        
        assert len(alerts) > 0
        assert any(a["type"] == "high_latency" for a in alerts)
        assert any(a["severity"] == "warning" for a in alerts)
    
    def test_check_health_high_error_rate(self):
        """Test health check detects high error rate."""
        monitor = ModelMonitor(error_rate_threshold=0.05)
        
        metrics = {
            "models": {
                "test_model": {
                    "avg_inference_time_ms": 10.0,
                    "p99_inference_time_ms": 15.0,
                    "success_rate": 0.90  # 10% error rate
                }
            }
        }
        
        alerts = monitor.check_health(metrics)
        
        assert len(alerts) > 0
        assert any(a["type"] == "high_error_rate" for a in alerts)
        assert any(a["severity"] == "critical" for a in alerts)
    
    def test_check_health_high_p99_latency(self):
        """Test health check detects high p99 latency."""
        monitor = ModelMonitor(alert_threshold_ms=10.0)
        
        metrics = {
            "models": {
                "test_model": {
                    "avg_inference_time_ms": 8.0,
                    "p99_inference_time_ms": 30.0,  # > 2x threshold
                    "success_rate": 0.99
                }
            }
        }
        
        alerts = monitor.check_health(metrics)
        
        assert len(alerts) > 0
        assert any(a["type"] == "high_p99_latency" for a in alerts)
    
    def test_check_health_multiple_models(self):
        """Test health check with multiple models."""
        monitor = ModelMonitor(alert_threshold_ms=10.0)
        
        metrics = {
            "models": {
                "model_a": {
                    "avg_inference_time_ms": 5.0,
                    "p99_inference_time_ms": 8.0,
                    "success_rate": 0.99
                },
                "model_b": {
                    "avg_inference_time_ms": 15.0,  # High
                    "p99_inference_time_ms": 20.0,
                    "success_rate": 0.99
                }
            }
        }
        
        alerts = monitor.check_health(metrics)
        
        assert len(alerts) > 0
        assert any(a["model"] == "model_b" for a in alerts)
        assert not any(a["model"] == "model_a" for a in alerts)
    
    def test_alert_accumulation(self):
        """Test alerts accumulate over time."""
        monitor = ModelMonitor(alert_threshold_ms=10.0)
        
        metrics = {
            "models": {
                "test_model": {
                    "avg_inference_time_ms": 15.0,
                    "p99_inference_time_ms": 20.0,
                    "success_rate": 0.99
                }
            }
        }
        
        # Generate alerts twice
        monitor.check_health(metrics)
        monitor.check_health(metrics)
        
        # Alerts should accumulate
        assert len(monitor.alerts) > 0
    
    def test_get_recent_alerts(self):
        """Test getting recent alerts."""
        import time
        from datetime import datetime
        
        monitor = ModelMonitor()
        
        # Add some alerts
        monitor.alerts = [
            {
                "severity": "warning",
                "model": "test_model",
                "type": "high_latency",
                "message": "Test alert",
                "timestamp": datetime.utcnow().isoformat()
            }
        ]
        
        recent = monitor.get_recent_alerts(hours=24)
        
        assert len(recent) > 0
    
    def test_alert_fields(self):
        """Test alert contains required fields."""
        monitor = ModelMonitor(alert_threshold_ms=10.0)
        
        metrics = {
            "models": {
                "test_model": {
                    "avg_inference_time_ms": 15.0,
                    "p99_inference_time_ms": 20.0,
                    "success_rate": 0.99
                }
            }
        }
        
        alerts = monitor.check_health(metrics)
        
        assert len(alerts) > 0
        alert = alerts[0]
        
        assert "severity" in alert
        assert "model" in alert
        assert "type" in alert
        assert "message" in alert
        assert "timestamp" in alert


@pytest.mark.monitoring
@pytest.mark.asyncio
class TestMetricsIntegration:
    """Test metrics integration."""
    
    async def test_end_to_end_metrics(self):
        """Test complete metrics workflow."""
        collector = MetricsCollector()
        monitor = ModelMonitor()
        
        # Simulate some inferences
        for i in range(100):
            await collector.record_inference(
                model_name="test_model",
                inference_time_ms=10.0 + i * 0.1,
                success=True
            )
        
        # Add some failures
        for _ in range(5):
            await collector.record_inference(
                model_name="test_model",
                inference_time_ms=0.0,
                success=False
            )
        
        # Get metrics
        metrics = collector.get_metrics()
        
        # Check health
        alerts = monitor.check_health(metrics)
        
        # Should have collected metrics
        assert "test_model" in metrics["models"]
        assert metrics["models"]["test_model"]["total_requests"] == 105
        
        # May or may not have alerts depending on thresholds
        assert isinstance(alerts, list)
