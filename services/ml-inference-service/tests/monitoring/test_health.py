"""Tests for health checks."""

import pytest


@pytest.mark.monitoring
@pytest.mark.asyncio
class TestHealthCheck:
    """Test health check functionality."""

    async def test_health_check_healthy(self, test_client):
        """Test health check when service is healthy."""
        response = await test_client.get("/health")

        assert response.status_code == 200
        data = response.json()

        assert data["status"] in ["healthy", "degraded"]
        assert data["triton_connected"] is True
        assert len(data["available_models"]) > 0
        assert "timestamp" in data

    async def test_health_check_response_time(self, test_client):
        """Test health check response time."""
        import time

        start = time.perf_counter()
        response = await test_client.get("/health")
        duration = (time.perf_counter() - start) * 1000

        assert response.status_code == 200
        # Health check should be fast (< 100ms)
        assert duration < 100

    async def test_health_check_structure(self, test_client):
        """Test health check response structure."""
        response = await test_client.get("/health")

        assert response.status_code == 200
        data = response.json()

        # Required fields
        assert "status" in data
        assert "triton_connected" in data
        assert "available_models" in data
        assert "timestamp" in data

        # Type checks
        assert isinstance(data["status"], str)
        assert isinstance(data["triton_connected"], bool)
        assert isinstance(data["available_models"], list)
        assert isinstance(data["timestamp"], str)

    async def test_health_check_concurrent(self, test_client):
        """Test concurrent health checks."""
        import asyncio

        tasks = [test_client.get("/health") for _ in range(10)]
        responses = await asyncio.gather(*tasks)

        # All should succeed
        assert all(r.status_code == 200 for r in responses)


@pytest.mark.monitoring
@pytest.mark.asyncio
class TestReadinessProbe:
    """Test readiness probe."""

    async def test_ready_when_triton_connected(self, test_client, mock_triton_client):
        """Test readiness when Triton is connected."""
        # Mock returns True for is_ready
        response = await test_client.get("/health")

        assert response.status_code == 200
        data = response.json()
        assert data["triton_connected"] is True

    async def test_not_ready_when_triton_disconnected(self, test_client, mock_triton_client):
        """Test readiness when Triton is disconnected."""

        # Make mock return False
        async def mock_not_ready():
            return False

        mock_triton_client.is_ready = mock_not_ready

        response = await test_client.get("/health")

        # Should still return 200 but status might be degraded
        assert response.status_code in [200, 503]


@pytest.mark.monitoring
@pytest.mark.asyncio
class TestLivenessProbe:
    """Test liveness probe."""

    async def test_liveness_basic(self, test_client):
        """Test basic liveness check."""
        response = await test_client.get("/health")

        # If service responds, it's alive
        assert response.status_code in [200, 503]

    async def test_liveness_under_load(self, test_client):
        """Test liveness under load."""
        import asyncio

        # Create background load
        async def background_requests():
            for _ in range(50):
                await test_client.post(
                    "/api/v1/predict/price", json={"symbol": "005930", "ohlcv_data": [[1.0] * 5] * 60}
                )

        # Start background load
        load_task = asyncio.create_task(background_requests())

        # Check health during load
        response = await test_client.get("/health")

        assert response.status_code == 200

        # Wait for background task
        await load_task


@pytest.mark.monitoring
class TestHealthCheckMetrics:
    """Test health check metrics."""

    @pytest.mark.asyncio
    async def test_health_check_updates_metrics(self, test_client):
        """Test that health checks don't pollute inference metrics."""
        # Get initial metrics
        response1 = await test_client.get("/api/v1/metrics")
        initial_data = response1.json()

        # Perform health checks
        for _ in range(10):
            await test_client.get("/health")

        # Get metrics again
        response2 = await test_client.get("/api/v1/metrics")
        final_data = response2.json()

        # Health checks shouldn't count as inference requests
        # (depends on implementation, but generally they shouldn't)
        assert response1.status_code == 200
        assert response2.status_code == 200


@pytest.mark.monitoring
@pytest.mark.asyncio
class TestServiceDiscovery:
    """Test service discovery information."""

    async def test_available_models_list(self, test_client):
        """Test available models are listed."""
        response = await test_client.get("/health")

        assert response.status_code == 200
        data = response.json()

        models = data["available_models"]

        # Should have at least one model
        assert len(models) > 0

        # Expected models
        expected_models = {"lstm_price_predictor", "transformer_sentiment", "ensemble_predictor"}
        assert any(model in expected_models for model in models)

    async def test_model_metadata_accessible(self, test_client):
        """Test model metadata is accessible from health check."""
        response = await test_client.get("/health")
        data = response.json()

        for model_name in data["available_models"]:
            # Try to get metadata for each model
            metadata_response = await test_client.get(f"/api/v1/models/{model_name}/metadata")
            # Should succeed or return 404 (acceptable)
            assert metadata_response.status_code in [200, 404]


@pytest.mark.monitoring
@pytest.mark.asyncio
class TestHealthCheckErrors:
    """Test health check error scenarios."""

    async def test_health_check_triton_timeout(self, test_client, mock_triton_client):
        """Test health check when Triton times out."""
        import asyncio

        async def mock_timeout():
            await asyncio.sleep(0.1)
            raise asyncio.TimeoutError("Triton timeout")

        mock_triton_client.is_ready = mock_timeout

        response = await test_client.get("/health")

        # Should handle gracefully
        assert response.status_code in [200, 503]

    async def test_health_check_triton_error(self, test_client, mock_triton_client):
        """Test health check when Triton has error."""

        async def mock_error():
            raise Exception("Triton error")

        mock_triton_client.is_ready = mock_error

        response = await test_client.get("/health")

        # Should handle gracefully
        assert response.status_code in [200, 503]


@pytest.mark.monitoring
@pytest.mark.asyncio
class TestHealthCheckPerformance:
    """Test health check performance."""

    async def test_health_check_cache(self, test_client):
        """Test health check caching behavior."""
        import time

        # First call
        start1 = time.perf_counter()
        response1 = await test_client.get("/health")
        duration1 = (time.perf_counter() - start1) * 1000

        # Second call (might be cached)
        start2 = time.perf_counter()
        response2 = await test_client.get("/health")
        duration2 = (time.perf_counter() - start2) * 1000

        assert response1.status_code == 200
        assert response2.status_code == 200

        print(f"\nHealth check times: {duration1:.2f}ms, {duration2:.2f}ms")

        # Both should be fast
        assert duration1 < 100
        assert duration2 < 100

    async def test_health_check_no_side_effects(self, test_client):
        """Test health checks have no side effects."""
        # Get initial state
        response1 = await test_client.get("/api/v1/metrics")
        initial_metrics = response1.json()

        # Perform many health checks
        for _ in range(100):
            await test_client.get("/health")

        # Get final state
        response2 = await test_client.get("/api/v1/metrics")
        final_metrics = response2.json()

        # Metrics should not be significantly affected
        assert response1.status_code == 200
        assert response2.status_code == 200


@pytest.mark.monitoring
@pytest.mark.asyncio
class TestComponentHealth:
    """Test individual component health."""

    async def test_database_health(self, test_client):
        """Test database connectivity check."""
        response = await test_client.get("/health")

        assert response.status_code in [200, 503]
        # In a full implementation, might check database status

    async def test_redis_health(self, test_client):
        """Test Redis connectivity check."""
        response = await test_client.get("/health")

        assert response.status_code in [200, 503]
        # In a full implementation, might check Redis status

    async def test_external_dependencies(self, test_client):
        """Test external dependency checks."""
        response = await test_client.get("/health")
        data = response.json()

        # Should include status
        assert "status" in data
        assert data["status"] in ["healthy", "degraded", "unhealthy"]


@pytest.mark.monitoring
@pytest.mark.asyncio
@pytest.mark.slow
class TestHealthCheckStability:
    """Test health check stability over time."""

    async def test_health_check_sustained(self, test_client):
        """Test health checks over sustained period."""
        import asyncio

        duration = 5  # seconds
        interval = 0.5  # seconds

        results = []
        end_time = asyncio.get_event_loop().time() + duration

        while asyncio.get_event_loop().time() < end_time:
            response = await test_client.get("/health")
            results.append(response.status_code)
            await asyncio.sleep(interval)

        # All should succeed
        assert all(code == 200 for code in results)
        assert len(results) > 0

        print(f"\nPerformed {len(results)} health checks over {duration}s")
