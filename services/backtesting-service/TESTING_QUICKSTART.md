# Testing Quick Start Guide

## 🚀 Quick Setup

```bash
# Navigate to service directory
cd services/backtesting-service

# Install dependencies
uv sync

# Run all tests
pytest

# Run with coverage
pytest --cov=src --cov-report=html --cov-report=term-missing
```

## 📋 Common Commands

### Run Specific Test Categories
```bash
# Unit tests only
pytest -m unit

# Integration tests
pytest -m integration

# Performance benchmarks
pytest -m performance

# Validation tests
pytest -m validation

# Skip slow tests
pytest -m "not slow"
```

### Run Specific Files
```bash
# Test event loop
pytest tests/unit/test_event_loop.py

# Test matching engine
pytest tests/unit/test_matching_engine.py

# Test portfolio
pytest tests/unit/test_portfolio.py

# Test performance metrics
pytest tests/unit/test_performance_metrics.py

# Test risk metrics
pytest tests/unit/test_risk_metrics.py

# Integration tests
pytest tests/integration/test_backtest_e2e.py
```

### Run Specific Tests
```bash
# Run one test function
pytest tests/unit/test_event_loop.py::TestEventLoop::test_event_creation

# Run one test class
pytest tests/unit/test_event_loop.py::TestEventLoop

# Run tests matching pattern
pytest -k "test_calculate_cagr"
```

### Debugging
```bash
# Show print statements
pytest -s

# Extra verbose
pytest -vv

# Show local variables on failure
pytest -l

# Stop on first failure
pytest -x

# Drop to debugger on failure
pytest --pdb
```

### Coverage
```bash
# Generate coverage report
pytest --cov=src --cov-report=html --cov-report=term-missing

# View HTML report
open htmlcov/index.html

# Show missing lines
pytest --cov=src --cov-report=term-missing

# Coverage for specific module
pytest --cov=src.engine.event_loop tests/unit/test_event_loop.py
```

## 📊 Expected Results

### Test Summary
```
=============== test session starts ================
platform darwin -- Python 3.11.x
plugins: cov-x.x.x, asyncio-x.x.x
collected 180 items

tests/unit/test_event_loop.py ...................... [24 PASSED]
tests/unit/test_matching_engine.py ................. [23 PASSED]
tests/unit/test_portfolio.py ...................... [32 PASSED]
tests/unit/test_performance_metrics.py ............. [23 PASSED]
tests/unit/test_risk_metrics.py ................... [20 PASSED]
tests/unit/test_data_loader.py .................... [16 PASSED]
tests/integration/test_backtest_e2e.py ............ [8 PASSED]
tests/performance/test_throughput.py .............. [6 PASSED]
tests/performance/test_metrics_performance.py ..... [13 PASSED]
tests/validation/test_metrics_validation.py ....... [15 PASSED]

=============== 180 passed in 25.43s ===============
```

### Coverage Summary
```
---------- coverage: platform darwin -----------
Name                                    Stmts   Miss  Cover
-----------------------------------------------------------
src/engine/event_loop.py                   45      2    96%
src/engine/matching_engine.py              68      5    93%
src/engine/portfolio.py                    95      5    95%
src/metrics/performance_metrics.py         82      0   100%
src/metrics/risk_metrics.py                75      0   100%
src/data/data_loader.py                    35      4    89%
-----------------------------------------------------------
TOTAL                                     413     16    96%
```

## 🎯 Test Categories

### Unit Tests (138 tests)
- `test_event_loop.py` - 24 tests
- `test_matching_engine.py` - 23 tests
- `test_portfolio.py` - 32 tests
- `test_performance_metrics.py` - 23 tests
- `test_risk_metrics.py` - 20 tests
- `test_data_loader.py` - 16 tests

### Integration Tests (8 tests)
- `test_backtest_e2e.py` - Complete workflow tests

### Performance Tests (19 tests)
- `test_throughput.py` - Throughput benchmarks
- `test_metrics_performance.py` - Metrics performance

### Validation Tests (15 tests)
- `test_metrics_validation.py` - Metrics accuracy validation

## 🔍 Test Markers

```bash
# Available markers
@pytest.mark.unit          # Unit tests
@pytest.mark.integration   # Integration tests
@pytest.mark.performance   # Performance benchmarks
@pytest.mark.validation    # Validation tests
@pytest.mark.slow          # Slow tests (> 1s)
```

## 📈 Performance Benchmarks

| Test | Target | Expected |
|------|--------|----------|
| 1 Year Daily Backtest | < 5s | ~2.3s |
| 1 Month Tick Backtest | < 30s | ~18.5s |
| CAGR (10k points) | < 100ms | ~45ms |
| MDD (10k points) | < 100ms | ~38ms |
| Sharpe (10k points) | < 100ms | ~52ms |
| Order Matching | > 1000/s | ~5234/s |

## 🛠️ Troubleshooting

### Tests Not Found
```bash
# Make sure you're in the right directory
cd services/backtesting-service

# Check pytest can find tests
pytest --collect-only
```

### Import Errors
```bash
# Install dependencies
uv sync

# Or with pip
pip install -e ".[dev]"
```

### Coverage Not Working
```bash
# Install pytest-cov
pip install pytest-cov

# Run with coverage
pytest --cov=src
```

### Slow Tests
```bash
# Skip slow tests
pytest -m "not slow"

# Run only fast tests
pytest -m unit -m "not slow"
```

## 📚 Additional Resources

- **Full Documentation**: `tests/README.md`
- **Test Summary**: `TEST_SUMMARY.md`
- **Source Code**: `src/`
- **Test Code**: `tests/`

## ✅ Pre-Commit Checklist

Before committing code:

```bash
# 1. Run all tests
pytest

# 2. Check coverage
pytest --cov=src --cov-report=term-missing

# 3. Run only fast tests if in a hurry
pytest -m "not slow"

# 4. Ensure no failures
# All tests should pass!
```

## 🎓 Writing New Tests

### Basic Test Template
```python
import pytest
from src.module import MyClass

class TestMyClass:
    """Test MyClass functionality"""

    def test_basic_operation(self):
        """Test basic operation succeeds"""
        # Arrange
        obj = MyClass(param=value)

        # Act
        result = obj.method()

        # Assert
        assert result == expected
```

### Using Fixtures
```python
def test_with_fixture(sample_tick_data):
    """Test using fixture"""
    assert len(sample_tick_data) > 0
```

### Adding Markers
```python
@pytest.mark.unit
def test_unit():
    """Unit test"""
    pass

@pytest.mark.slow
def test_slow_operation():
    """Slow test"""
    time.sleep(2)
```

## 📞 Getting Help

1. Check `tests/README.md` for detailed docs
2. Review existing tests for examples
3. Run with `-vv` for more details
4. Use `--pdb` to debug failures

---

**Happy Testing!** 🎉
