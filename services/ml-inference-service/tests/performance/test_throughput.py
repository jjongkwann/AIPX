"""Performance tests for throughput."""
import pytest
import asyncio
import time
from statistics import mean


@pytest.mark.performance
@pytest.mark.asyncio
class TestThroughput:
    """Test inference throughput."""
    
    async def test_lstm_throughput(self, mock_triton_client, sample_features):
        """Test LSTM throughput (target: > 500 req/s)."""
        n_requests = 1000
        start_time = time.perf_counter()
        
        tasks = [
            mock_triton_client.predict_price(sample_features)
            for _ in range(n_requests)
        ]
        
        await asyncio.gather(*tasks)
        
        duration = time.perf_counter() - start_time
        throughput = n_requests / duration
        
        print(f"\nLSTM Throughput:")
        print(f"  Requests: {n_requests}")
        print(f"  Duration: {duration:.2f}s")
        print(f"  Throughput: {throughput:.2f} req/s")
        
        # Target: > 500 req/s (mock will be much higher)
        assert throughput > 100
    
    async def test_sentiment_throughput(self, mock_triton_client, sample_tokenized_input):
        """Test Sentiment throughput (target: > 300 req/s)."""
        n_requests = 1000
        input_ids = sample_tokenized_input["input_ids"]
        attention_mask = sample_tokenized_input["attention_mask"]
        
        start_time = time.perf_counter()
        
        tasks = [
            mock_triton_client.analyze_sentiment(input_ids, attention_mask)
            for _ in range(n_requests)
        ]
        
        await asyncio.gather(*tasks)
        
        duration = time.perf_counter() - start_time
        throughput = n_requests / duration
        
        print(f"\nSentiment Throughput:")
        print(f"  Requests: {n_requests}")
        print(f"  Duration: {duration:.2f}s")
        print(f"  Throughput: {throughput:.2f} req/s")
        
        assert throughput > 100
    
    async def test_batch_throughput(self, mock_triton_client):
        """Test batch inference throughput."""
        batch_sizes = [1, 4, 8, 16, 32]
        results = {}
        
        for batch_size in batch_sizes:
            n_batches = 100
            total_samples = n_batches * batch_size
            
            features_batch = [
                sample_features
                for sample_features in [
                    pytest.lazy_fixture('sample_features')
                ] * batch_size
            ]
            
            import numpy as np
            features_batch = [np.random.randn(60, 5).astype('float32') for _ in range(batch_size)]
            
            start_time = time.perf_counter()
            
            tasks = [
                mock_triton_client.batch_predict_prices(features_batch)
                for _ in range(n_batches)
            ]
            
            await asyncio.gather(*tasks)
            
            duration = time.perf_counter() - start_time
            throughput = total_samples / duration
            
            results[batch_size] = {
                "total_samples": total_samples,
                "duration": duration,
                "throughput": throughput
            }
        
        print(f"\nBatch Throughput:")
        for batch_size, metrics in results.items():
            print(f"  Batch {batch_size}: {metrics['throughput']:.2f} samples/s")
        
        # Larger batches should have higher throughput
        assert results[32]["throughput"] > results[1]["throughput"]
    
    @pytest.mark.slow
    async def test_sustained_throughput(self, mock_triton_client, sample_features):
        """Test sustained throughput over time."""
        duration_seconds = 10
        window_size = 1.0  # 1 second windows
        
        throughputs = []
        end_time = time.perf_counter() + duration_seconds
        
        while time.perf_counter() < end_time:
            window_start = time.perf_counter()
            window_end = window_start + window_size
            
            requests = 0
            while time.perf_counter() < window_end:
                await mock_triton_client.predict_price(sample_features)
                requests += 1
            
            window_duration = time.perf_counter() - window_start
            window_throughput = requests / window_duration
            throughputs.append(window_throughput)
        
        avg_throughput = mean(throughputs)
        min_throughput = min(throughputs)
        max_throughput = max(throughputs)
        
        print(f"\nSustained Throughput ({duration_seconds}s):")
        print(f"  Average: {avg_throughput:.2f} req/s")
        print(f"  Min: {min_throughput:.2f} req/s")
        print(f"  Max: {max_throughput:.2f} req/s")
        print(f"  Stability: {(min_throughput/avg_throughput)*100:.1f}%")
        
        # Throughput should be relatively stable
        assert min_throughput > avg_throughput * 0.5


@pytest.mark.performance
@pytest.mark.asyncio
class TestConcurrency:
    """Test concurrent request handling."""
    
    async def test_concurrent_clients(self, mock_triton_client, sample_features):
        """Test handling multiple concurrent clients."""
        n_clients = [1, 10, 50, 100]
        results = {}
        
        for n in n_clients:
            requests_per_client = 100
            
            async def client_workload():
                for _ in range(requests_per_client):
                    await mock_triton_client.predict_price(sample_features)
            
            start_time = time.perf_counter()
            
            tasks = [client_workload() for _ in range(n)]
            await asyncio.gather(*tasks)
            
            duration = time.perf_counter() - start_time
            total_requests = n * requests_per_client
            throughput = total_requests / duration
            
            results[n] = {
                "total_requests": total_requests,
                "duration": duration,
                "throughput": throughput
            }
        
        print(f"\nConcurrent Clients Throughput:")
        for n_clients, metrics in results.items():
            print(f"  {n_clients} clients: {metrics['throughput']:.2f} req/s")
        
        # System should scale with more clients (to a point)
        assert results[10]["throughput"] > results[1]["throughput"]
    
    async def test_mixed_workload(self, mock_triton_client, sample_features, sample_tokenized_input):
        """Test mixed workload (price + sentiment)."""
        n_requests = 500
        
        input_ids = sample_tokenized_input["input_ids"]
        attention_mask = sample_tokenized_input["attention_mask"]
        
        # Mix of request types
        async def price_request():
            return await mock_triton_client.predict_price(sample_features)
        
        async def sentiment_request():
            return await mock_triton_client.analyze_sentiment(input_ids, attention_mask)
        
        tasks = []
        for i in range(n_requests):
            if i % 2 == 0:
                tasks.append(price_request())
            else:
                tasks.append(sentiment_request())
        
        start_time = time.perf_counter()
        await asyncio.gather(*tasks)
        duration = time.perf_counter() - start_time
        
        throughput = n_requests / duration
        
        print(f"\nMixed Workload Throughput:")
        print(f"  Total requests: {n_requests}")
        print(f"  Duration: {duration:.2f}s")
        print(f"  Throughput: {throughput:.2f} req/s")
        
        assert throughput > 100


@pytest.mark.performance
@pytest.mark.asyncio
class TestResourceUtilization:
    """Test resource utilization."""
    
    async def test_cpu_utilization(self, mock_triton_client, sample_features):
        """Test CPU utilization during inference."""
        import psutil
        import os
        
        process = psutil.Process(os.getpid())
        
        # Measure CPU usage
        process.cpu_percent(interval=None)  # First call returns 0
        
        # Run inferences
        tasks = [
            mock_triton_client.predict_price(sample_features)
            for _ in range(1000)
        ]
        await asyncio.gather(*tasks)
        
        cpu_percent = process.cpu_percent(interval=1.0)
        
        print(f"\nCPU Utilization: {cpu_percent:.1f}%")
        
        # Should use some CPU
        assert cpu_percent > 0
    
    async def test_connection_pooling(self, mock_triton_client, sample_features):
        """Test connection pooling efficiency."""
        n_requests = 1000
        
        # Without pooling (new connection each time) - simulated
        start_time = time.perf_counter()
        for _ in range(n_requests):
            await mock_triton_client.predict_price(sample_features)
        sequential_duration = time.perf_counter() - start_time
        
        # With pooling (concurrent)
        start_time = time.perf_counter()
        tasks = [
            mock_triton_client.predict_price(sample_features)
            for _ in range(n_requests)
        ]
        await asyncio.gather(*tasks)
        concurrent_duration = time.perf_counter() - start_time
        
        print(f"\nConnection Efficiency:")
        print(f"  Sequential: {sequential_duration:.2f}s")
        print(f"  Concurrent: {concurrent_duration:.2f}s")
        print(f"  Speedup: {sequential_duration/concurrent_duration:.2f}x")
        
        # Concurrent should be faster
        assert concurrent_duration < sequential_duration


@pytest.mark.performance
@pytest.mark.asyncio
@pytest.mark.slow
class TestScalability:
    """Test system scalability."""
    
    async def test_linear_scalability(self, mock_triton_client, sample_features):
        """Test if throughput scales linearly with load."""
        load_levels = [100, 200, 500, 1000]
        results = {}
        
        for n_requests in load_levels:
            start_time = time.perf_counter()
            
            tasks = [
                mock_triton_client.predict_price(sample_features)
                for _ in range(n_requests)
            ]
            
            await asyncio.gather(*tasks)
            
            duration = time.perf_counter() - start_time
            throughput = n_requests / duration
            
            results[n_requests] = {
                "duration": duration,
                "throughput": throughput
            }
        
        print(f"\nScalability Test:")
        for n_requests, metrics in results.items():
            print(f"  {n_requests} requests: {metrics['throughput']:.2f} req/s")
        
        # Check if throughput remains relatively constant
        throughputs = [r["throughput"] for r in results.values()]
        avg_throughput = mean(throughputs)
        
        # All throughputs should be within 50% of average
        for tp in throughputs:
            assert tp > avg_throughput * 0.5
