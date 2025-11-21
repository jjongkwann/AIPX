"""Unit tests for LSTM model."""
import pytest
import torch
import torch.nn as nn
import numpy as np
import tempfile
import os

from src.models.lstm_model import (
    LSTMPredictor,
    AdvancedLSTMPredictor,
    EnsembleCombiner,
    create_lstm_model,
    save_model_for_triton
)


@pytest.mark.unit
class TestLSTMPredictor:
    """Test LSTMPredictor model."""
    
    def test_model_initialization(self):
        """Test model initialization with default parameters."""
        model = LSTMPredictor()
        assert model.input_size == 5
        assert model.hidden_size == 128
        assert model.num_layers == 2
        assert not model.bidirectional
    
    def test_model_initialization_custom(self):
        """Test model initialization with custom parameters."""
        model = LSTMPredictor(
            input_size=10,
            hidden_size=256,
            num_layers=3,
            dropout=0.3,
            bidirectional=True
        )
        assert model.input_size == 10
        assert model.hidden_size == 256
        assert model.num_layers == 3
        assert model.bidirectional
    
    def test_forward_pass(self, sample_torch_tensor):
        """Test forward pass with valid input."""
        model = LSTMPredictor()
        model.eval()
        
        with torch.no_grad():
            output = model(sample_torch_tensor)
        
        assert output.shape == (4, 1)  # batch_size=4, output_size=1
        assert not torch.isnan(output).any()
        assert not torch.isinf(output).any()
    
    def test_forward_pass_single_sample(self):
        """Test forward pass with single sample."""
        model = LSTMPredictor()
        model.eval()
        
        x = torch.randn(1, 60, 5)
        
        with torch.no_grad():
            output = model(x)
        
        assert output.shape == (1, 1)
    
    def test_forward_pass_different_batch_sizes(self):
        """Test with different batch sizes."""
        model = LSTMPredictor()
        model.eval()
        
        for batch_size in [1, 4, 8, 16, 32]:
            x = torch.randn(batch_size, 60, 5)
            with torch.no_grad():
                output = model(x)
            assert output.shape == (batch_size, 1)
    
    def test_model_parameters_count(self):
        """Test that model has expected number of parameters."""
        model = LSTMPredictor()
        total_params = sum(p.numel() for p in model.parameters())
        
        # Should have > 100k parameters
        assert total_params > 100000
    
    def test_model_trainable_parameters(self):
        """Test that all parameters are trainable."""
        model = LSTMPredictor()
        trainable_params = sum(p.numel() for p in model.parameters() if p.requires_grad)
        total_params = sum(p.numel() for p in model.parameters())
        
        assert trainable_params == total_params
    
    def test_model_gradient_flow(self):
        """Test that gradients flow properly."""
        model = LSTMPredictor()
        model.train()
        
        x = torch.randn(4, 60, 5, requires_grad=True)
        target = torch.randn(4, 1)
        
        output = model(x)
        loss = nn.MSELoss()(output, target)
        loss.backward()
        
        # Check that gradients are computed
        for param in model.parameters():
            if param.requires_grad:
                assert param.grad is not None
    
    def test_attention_weights(self):
        """Test attention weight extraction."""
        model = LSTMPredictor()
        model.eval()
        
        x = torch.randn(2, 60, 5)
        
        attention_weights = model.get_attention_weights(x)
        
        assert attention_weights.shape == (2, 60, 1)
        # Attention weights should sum to 1 for each sample
        assert torch.allclose(attention_weights.sum(dim=1), torch.ones(2, 1), atol=1e-5)
    
    def test_model_eval_mode(self):
        """Test model behavior in eval mode."""
        model = LSTMPredictor(dropout=0.5)
        model.eval()
        
        x = torch.randn(4, 60, 5)
        
        # Multiple forward passes should give same result
        with torch.no_grad():
            output1 = model(x)
            output2 = model(x)
        
        assert torch.allclose(output1, output2)
    
    def test_model_train_mode(self):
        """Test model behavior in train mode."""
        model = LSTMPredictor(dropout=0.5)
        model.train()
        
        x = torch.randn(4, 60, 5)
        
        # Multiple forward passes may give different results due to dropout
        output1 = model(x)
        output2 = model(x)
        
        # Outputs should be similar but not identical
        assert not torch.allclose(output1, output2, atol=1e-5)


@pytest.mark.unit
class TestAdvancedLSTMPredictor:
    """Test AdvancedLSTMPredictor model."""
    
    def test_advanced_model_initialization(self):
        """Test advanced model initialization."""
        model = AdvancedLSTMPredictor()
        assert model.input_size == 5
        assert model.hidden_size == 256
        assert model.num_layers == 3
    
    def test_advanced_forward_pass(self):
        """Test forward pass."""
        model = AdvancedLSTMPredictor()
        model.eval()
        
        x = torch.randn(4, 60, 5)
        
        with torch.no_grad():
            output = model(x)
        
        assert output.shape == (4, 1)
        assert not torch.isnan(output).any()
    
    def test_advanced_model_larger(self):
        """Test that advanced model is larger than basic."""
        basic_model = LSTMPredictor()
        advanced_model = AdvancedLSTMPredictor()
        
        basic_params = sum(p.numel() for p in basic_model.parameters())
        advanced_params = sum(p.numel() for p in advanced_model.parameters())
        
        assert advanced_params > basic_params
    
    def test_residual_connections(self):
        """Test residual connections work properly."""
        model = AdvancedLSTMPredictor(num_layers=3)
        model.eval()
        
        x = torch.randn(2, 60, 5)
        
        with torch.no_grad():
            output = model(x)
        
        # Should produce reasonable outputs
        assert not torch.isnan(output).any()
        assert not torch.isinf(output).any()


@pytest.mark.unit
class TestEnsembleCombiner:
    """Test EnsembleCombiner model."""
    
    def test_ensemble_initialization(self):
        """Test ensemble combiner initialization."""
        model = EnsembleCombiner()
        assert model is not None
    
    def test_ensemble_forward(self):
        """Test ensemble forward pass."""
        model = EnsembleCombiner()
        model.eval()
        
        price_pred = torch.tensor([[71500.0], [71600.0]], dtype=torch.float32)
        sentiment = torch.tensor([[0.75], [0.80]], dtype=torch.float32)
        sentiment_conf = torch.tensor([[0.90], [0.85]], dtype=torch.float32)
        
        with torch.no_grad():
            final_pred, price_comp, sent_comp, confidence = model(
                price_pred, sentiment, sentiment_conf
            )
        
        assert final_pred.shape == (2, 1)
        assert price_comp.shape == (2, 1)
        assert sent_comp.shape == (2, 1)
        assert confidence.shape == (2, 1)
        
        # Confidence should be in [0, 1]
        assert torch.all(confidence >= 0)
        assert torch.all(confidence <= 1)
    
    def test_ensemble_confidence_range(self):
        """Test that confidence is always in valid range."""
        model = EnsembleCombiner()
        model.eval()
        
        # Test with extreme values
        price_pred = torch.tensor([[100000.0]], dtype=torch.float32)
        sentiment = torch.tensor([[1.0]], dtype=torch.float32)
        sentiment_conf = torch.tensor([[1.0]], dtype=torch.float32)
        
        with torch.no_grad():
            _, _, _, confidence = model(price_pred, sentiment, sentiment_conf)
        
        assert 0 <= confidence.item() <= 1


@pytest.mark.unit
class TestModelFactory:
    """Test model factory functions."""
    
    def test_create_basic_lstm(self):
        """Test creating basic LSTM model."""
        config = {
            'type': 'basic',
            'input_size': 5,
            'hidden_size': 128,
            'num_layers': 2
        }
        
        model = create_lstm_model(config)
        
        assert isinstance(model, LSTMPredictor)
        assert model.input_size == 5
        assert model.hidden_size == 128
    
    def test_create_advanced_lstm(self):
        """Test creating advanced LSTM model."""
        config = {
            'type': 'advanced',
            'input_size': 5,
            'hidden_size': 256,
            'num_layers': 3
        }
        
        model = create_lstm_model(config)
        
        assert isinstance(model, AdvancedLSTMPredictor)
        assert model.input_size == 5
        assert model.hidden_size == 256
    
    def test_create_unknown_type(self):
        """Test creating model with unknown type."""
        config = {'type': 'unknown'}
        
        with pytest.raises(ValueError, match="Unknown model type"):
            create_lstm_model(config)


@pytest.mark.unit
class TestModelSerialization:
    """Test model serialization."""
    
    def test_save_model_scripting(self):
        """Test saving model with TorchScript."""
        model = LSTMPredictor()
        model.eval()
        
        with tempfile.TemporaryDirectory() as tmpdir:
            save_path = os.path.join(tmpdir, "model.pt")
            save_model_for_triton(model, save_path)
            
            assert os.path.exists(save_path)
            
            # Try loading
            loaded_model = torch.jit.load(save_path)
            assert loaded_model is not None
    
    def test_save_model_tracing(self):
        """Test saving model with tracing."""
        model = LSTMPredictor()
        model.eval()
        
        example_input = torch.randn(1, 60, 5)
        
        with tempfile.TemporaryDirectory() as tmpdir:
            save_path = os.path.join(tmpdir, "model.pt")
            save_model_for_triton(model, save_path, example_input)
            
            assert os.path.exists(save_path)
            
            # Load and test
            loaded_model = torch.jit.load(save_path)
            
            with torch.no_grad():
                original_output = model(example_input)
                loaded_output = loaded_model(example_input)
            
            assert torch.allclose(original_output, loaded_output, atol=1e-5)
    
    def test_load_and_infer(self):
        """Test loading model and running inference."""
        model = LSTMPredictor()
        model.eval()
        
        example_input = torch.randn(2, 60, 5)
        
        with tempfile.TemporaryDirectory() as tmpdir:
            save_path = os.path.join(tmpdir, "model.pt")
            save_model_for_triton(model, save_path, example_input)
            
            loaded_model = torch.jit.load(save_path)
            loaded_model.eval()
            
            with torch.no_grad():
                output = loaded_model(example_input)
            
            assert output.shape == (2, 1)
            assert not torch.isnan(output).any()


@pytest.mark.unit
@pytest.mark.gpu
class TestModelGPU:
    """Test model on GPU (if available)."""
    
    @pytest.mark.skipif(not torch.cuda.is_available(), reason="CUDA not available")
    def test_model_on_gpu(self):
        """Test model inference on GPU."""
        model = LSTMPredictor()
        model = model.cuda()
        model.eval()
        
        x = torch.randn(4, 60, 5).cuda()
        
        with torch.no_grad():
            output = model(x)
        
        assert output.is_cuda
        assert output.shape == (4, 1)
    
    @pytest.mark.skipif(not torch.cuda.is_available(), reason="CUDA not available")
    def test_model_gpu_cpu_consistency(self):
        """Test that GPU and CPU give same results."""
        model = LSTMPredictor()
        model.eval()
        
        x = torch.randn(2, 60, 5)
        
        # CPU inference
        with torch.no_grad():
            cpu_output = model(x)
        
        # GPU inference
        model_gpu = model.cuda()
        with torch.no_grad():
            gpu_output = model_gpu(x.cuda())
        
        # Results should be very close
        assert torch.allclose(cpu_output, gpu_output.cpu(), atol=1e-4)
