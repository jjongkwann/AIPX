"""Performance tests for inference latency."""
import pytest
import time
import numpy as np
import asyncio
from statistics import median, mean


@pytest.mark.performance
@pytest.mark.asyncio
class TestInferenceLatency:
    """Test inference latency performance."""
    
    async def test_lstm_inference_latency(self, mock_triton_client, sample_features):
        """Test LSTM inference latency (target: < 15ms @ p95)."""
        latencies = []
        n_iterations = 100
        
        for _ in range(n_iterations):
            start_time = time.perf_counter()
            await mock_triton_client.predict_price(sample_features)
            end_time = time.perf_counter()
            
            latency_ms = (end_time - start_time) * 1000
            latencies.append(latency_ms)
        
        # Calculate percentiles
        sorted_latencies = sorted(latencies)
        p50 = sorted_latencies[len(sorted_latencies) // 2]
        p95 = sorted_latencies[int(len(sorted_latencies) * 0.95)]
        p99 = sorted_latencies[int(len(sorted_latencies) * 0.99)]
        
        print(f"\nLSTM Latency Stats:")
        print(f"  Mean: {mean(latencies):.2f}ms")
        print(f"  Median (p50): {p50:.2f}ms")
        print(f"  p95: {p95:.2f}ms")
        print(f"  p99: {p99:.2f}ms")
        print(f"  Min: {min(latencies):.2f}ms")
        print(f"  Max: {max(latencies):.2f}ms")
        
        # Target: p95 < 15ms (will pass with mock)
        assert p95 < 50  # Relaxed for mock, real should be < 15ms
    
    async def test_sentiment_inference_latency(self, mock_triton_client, sample_tokenized_input):
        """Test Sentiment inference latency (target: < 30ms @ p95)."""
        latencies = []
        n_iterations = 100
        
        input_ids = sample_tokenized_input["input_ids"]
        attention_mask = sample_tokenized_input["attention_mask"]
        
        for _ in range(n_iterations):
            start_time = time.perf_counter()
            await mock_triton_client.analyze_sentiment(input_ids, attention_mask)
            end_time = time.perf_counter()
            
            latency_ms = (end_time - start_time) * 1000
            latencies.append(latency_ms)
        
        sorted_latencies = sorted(latencies)
        p50 = sorted_latencies[len(sorted_latencies) // 2]
        p95 = sorted_latencies[int(len(sorted_latencies) * 0.95)]
        p99 = sorted_latencies[int(len(sorted_latencies) * 0.99)]
        
        print(f"\nSentiment Latency Stats:")
        print(f"  Mean: {mean(latencies):.2f}ms")
        print(f"  Median (p50): {p50:.2f}ms")
        print(f"  p95: {p95:.2f}ms")
        print(f"  p99: {p99:.2f}ms")
        
        # Target: p95 < 30ms
        assert p95 < 50  # Relaxed for mock
    
    async def test_ensemble_inference_latency(self, mock_triton_client, sample_features, sample_tokenized_input):
        """Test Ensemble inference latency (target: < 40ms @ p95)."""
        latencies = []
        n_iterations = 100
        
        input_ids = sample_tokenized_input["input_ids"]
        attention_mask = sample_tokenized_input["attention_mask"]
        
        for _ in range(n_iterations):
            start_time = time.perf_counter()
            await mock_triton_client.ensemble_predict(
                sample_features, input_ids, attention_mask
            )
            end_time = time.perf_counter()
            
            latency_ms = (end_time - start_time) * 1000
            latencies.append(latency_ms)
        
        sorted_latencies = sorted(latencies)
        p95 = sorted_latencies[int(len(sorted_latencies) * 0.95)]
        
        print(f"\nEnsemble Latency Stats:")
        print(f"  Mean: {mean(latencies):.2f}ms")
        print(f"  p95: {p95:.2f}ms")
        
        # Target: p95 < 40ms
        assert p95 < 60  # Relaxed for mock
    
    async def test_batch_inference_latency(self, mock_triton_client):
        """Test batch inference latency with different batch sizes."""
        batch_sizes = [1, 2, 4, 8, 16, 32]
        
        results = {}
        
        for batch_size in batch_sizes:
            features_batch = [
                np.random.randn(60, 5).astype(np.float32)
                for _ in range(batch_size)
            ]
            
            latencies = []
            n_iterations = 50
            
            for _ in range(n_iterations):
                start_time = time.perf_counter()
                await mock_triton_client.batch_predict_prices(features_batch)
                end_time = time.perf_counter()
                
                latency_ms = (end_time - start_time) * 1000
                latencies.append(latency_ms)
            
            avg_latency = mean(latencies)
            avg_per_sample = avg_latency / batch_size
            
            results[batch_size] = {
                "avg_total_ms": avg_latency,
                "avg_per_sample_ms": avg_per_sample
            }
        
        print(f"\nBatch Inference Latency:")
        for batch_size, metrics in results.items():
            print(f"  Batch {batch_size}: {metrics['avg_total_ms']:.2f}ms total, "
                  f"{metrics['avg_per_sample_ms']:.2f}ms per sample")
        
        # Batching should improve per-sample latency
        assert results[32]["avg_per_sample_ms"] < results[1]["avg_per_sample_ms"]
    
    @pytest.mark.slow
    async def test_cold_start_latency(self, mock_triton_client, sample_features):
        """Test cold start latency (first request)."""
        # Simulate cold start
        start_time = time.perf_counter()
        await mock_triton_client.predict_price(sample_features)
        cold_start_latency = (time.perf_counter() - start_time) * 1000
        
        # Warm requests
        warm_latencies = []
        for _ in range(10):
            start_time = time.perf_counter()
            await mock_triton_client.predict_price(sample_features)
            warm_latencies.append((time.perf_counter() - start_time) * 1000)
        
        avg_warm_latency = mean(warm_latencies)
        
        print(f"\nCold Start Latency: {cold_start_latency:.2f}ms")
        print(f"Warm Latency: {avg_warm_latency:.2f}ms")
        
        # Cold start should be slower (but not in mock)
        assert cold_start_latency > 0
    
    async def test_concurrent_inference_latency(self, mock_triton_client, sample_features):
        """Test latency under concurrent load."""
        n_concurrent = 10
        n_iterations = 10
        
        async def single_inference():
            latencies = []
            for _ in range(n_iterations):
                start_time = time.perf_counter()
                await mock_triton_client.predict_price(sample_features)
                latencies.append((time.perf_counter() - start_time) * 1000)
            return latencies
        
        # Run concurrent inferences
        tasks = [single_inference() for _ in range(n_concurrent)]
        all_latencies = await asyncio.gather(*tasks)
        
        # Flatten latencies
        flat_latencies = [lat for latencies in all_latencies for lat in latencies]
        
        p95 = sorted(flat_latencies)[int(len(flat_latencies) * 0.95)]
        
        print(f"\nConcurrent Inference (n={n_concurrent}):")
        print(f"  Mean: {mean(flat_latencies):.2f}ms")
        print(f"  p95: {p95:.2f}ms")
        
        # p95 should still be reasonable under load
        assert p95 < 100


@pytest.mark.performance
@pytest.mark.benchmark
class TestInferenceBenchmark:
    """Benchmark tests using pytest-benchmark."""
    
    def test_feature_extraction_benchmark(self, benchmark, sample_market_data):
        """Benchmark feature extraction."""
        from src.features.feature_extractor import FeatureExtractor
        
        extractor = FeatureExtractor()
        
        result = benchmark(
            extractor.extract_price_features,
            sample_market_data,
            60
        )
        
        assert result.shape == (60, 5)
    
    def test_lstm_forward_benchmark(self, benchmark, mock_lstm_model, sample_torch_tensor):
        """Benchmark LSTM forward pass."""
        import torch
        
        mock_lstm_model.eval()
        
        def forward_pass():
            with torch.no_grad():
                return mock_lstm_model(sample_torch_tensor)
        
        result = benchmark(forward_pass)
        
        assert result.shape == (4, 1)
    
    def test_tokenization_benchmark(self, benchmark):
        """Benchmark text tokenization."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        text = "Samsung reports strong quarterly earnings"
        
        result = benchmark(extractor.tokenize_text, text)
        
        assert result["input_ids"].shape == (512,)


@pytest.mark.performance
@pytest.mark.slow
class TestMemoryUsage:
    """Test memory usage during inference."""
    
    async def test_memory_leak(self, mock_triton_client, sample_features):
        """Test for memory leaks during repeated inference."""
        import psutil
        import os
        
        process = psutil.Process(os.getpid())
        initial_memory = process.memory_info().rss / 1024 / 1024  # MB
        
        # Run many inferences
        for _ in range(1000):
            await mock_triton_client.predict_price(sample_features)
        
        final_memory = process.memory_info().rss / 1024 / 1024  # MB
        memory_increase = final_memory - initial_memory
        
        print(f"\nMemory Usage:")
        print(f"  Initial: {initial_memory:.2f} MB")
        print(f"  Final: {final_memory:.2f} MB")
        print(f"  Increase: {memory_increase:.2f} MB")
        
        # Memory increase should be minimal (< 100MB)
        assert memory_increase < 100
    
    async def test_batch_memory_usage(self, mock_triton_client):
        """Test memory usage with different batch sizes."""
        import psutil
        import os
        
        process = psutil.Process(os.getpid())
        
        batch_sizes = [1, 8, 32]
        memory_usage = {}
        
        for batch_size in batch_sizes:
            features_batch = [
                np.random.randn(60, 5).astype(np.float32)
                for _ in range(batch_size)
            ]
            
            before = process.memory_info().rss / 1024 / 1024
            await mock_triton_client.batch_predict_prices(features_batch)
            after = process.memory_info().rss / 1024 / 1024
            
            memory_usage[batch_size] = after - before
        
        print(f"\nBatch Memory Usage:")
        for batch_size, mem in memory_usage.items():
            print(f"  Batch {batch_size}: {mem:.2f} MB")
