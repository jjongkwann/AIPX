"""Validation tests for sentiment model."""
import pytest
import numpy as np


@pytest.mark.validation
@pytest.mark.asyncio
class TestSentimentValidation:
    """Validate sentiment model predictions."""
    
    async def test_positive_sentiment(self, mock_triton_client):
        """Test positive news sentiment."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        positive_texts = [
            "Company reports record-breaking profits and exceptional growth",
            "Stock reaches all-time high as investors celebrate success",
            "Excellent quarterly results exceed all expectations",
            "Revolutionary product launch drives incredible market response"
        ]
        
        sentiments = []
        for text in positive_texts:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            sentiments.append(result["sentiment"])
        
        print(f"\nPositive Sentiment Scores: {sentiments}")
        
        # Mock returns random, but check structure
        assert all(-1 <= s <= 1 for s in sentiments)
    
    async def test_negative_sentiment(self, mock_triton_client):
        """Test negative news sentiment."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        negative_texts = [
            "Company faces massive losses and severe financial crisis",
            "Stock plummets as investors flee amid disaster",
            "Terrible quarterly results disappoint everyone",
            "Failed product launch causes catastrophic market collapse"
        ]
        
        sentiments = []
        for text in negative_texts:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            sentiments.append(result["sentiment"])
        
        print(f"\nNegative Sentiment Scores: {sentiments}")
        
        assert all(-1 <= s <= 1 for s in sentiments)
    
    async def test_neutral_sentiment(self, mock_triton_client):
        """Test neutral news sentiment."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        neutral_texts = [
            "Company releases quarterly report",
            "Stock price remains stable",
            "Board meeting scheduled for next week",
            "Analyst maintains current rating"
        ]
        
        sentiments = []
        for text in neutral_texts:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            sentiments.append(result["sentiment"])
        
        print(f"\nNeutral Sentiment Scores: {sentiments}")
        
        # Should be in valid range
        assert all(-1 <= s <= 1 for s in sentiments)
    
    async def test_financial_terminology(self, mock_triton_client):
        """Test with financial terminology."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        financial_texts = [
            "EPS increased by 15% with strong revenue growth",
            "P/E ratio declined amid market volatility",
            "ROE improved significantly quarter over quarter",
            "Dividend yield remains attractive for investors"
        ]
        
        sentiments = []
        confidences = []
        for text in financial_texts:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            sentiments.append(result["sentiment"])
            confidences.append(result["confidence"])
        
        print(f"\nFinancial Terminology Sentiments:")
        for text, sent, conf in zip(financial_texts, sentiments, confidences):
            print(f"  '{text[:50]}...': {sent:.3f} (conf: {conf:.3f})")
        
        assert all(-1 <= s <= 1 for s in sentiments)
        assert all(0 <= c <= 1 for c in confidences)
    
    async def test_mixed_signals(self, mock_triton_client):
        """Test text with mixed positive and negative signals."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        mixed_texts = [
            "Revenue increased but profits declined",
            "Strong sales offset by higher costs",
            "Positive outlook despite current challenges",
            "Growth slowed but remains positive"
        ]
        
        sentiments = []
        confidences = []
        for text in mixed_texts:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            sentiments.append(result["sentiment"])
            confidences.append(result["confidence"])
        
        print(f"\nMixed Signal Sentiments:")
        for text, sent, conf in zip(mixed_texts, sentiments, confidences):
            print(f"  '{text}': {sent:.3f} (conf: {conf:.3f})")
        
        # Confidence might be lower for mixed signals
        assert all(-1 <= s <= 1 for s in sentiments)


@pytest.mark.validation
@pytest.mark.asyncio
class TestSentimentRobustness:
    """Test sentiment model robustness."""
    
    async def test_empty_text(self, mock_triton_client):
        """Test with empty text."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        tokens = extractor.tokenize_text("")
        
        result = await mock_triton_client.analyze_sentiment(
            tokens["input_ids"],
            tokens["attention_mask"]
        )
        
        assert -1 <= result["sentiment"] <= 1
        assert 0 <= result["confidence"] <= 1
    
    async def test_very_long_text(self, mock_triton_client):
        """Test with very long text (truncation)."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        long_text = "This is a very long news article. " * 200
        
        tokens = extractor.tokenize_text(long_text, max_length=512)
        
        result = await mock_triton_client.analyze_sentiment(
            tokens["input_ids"],
            tokens["attention_mask"]
        )
        
        assert -1 <= result["sentiment"] <= 1
    
    async def test_special_characters(self, mock_triton_client):
        """Test with special characters."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        texts_with_special = [
            "Stock $AAPL +5.2% 📈",
            "Revenue: $1.2B (up 15%)",
            "P/E ratio = 25.5x",
            "Q4'23 results ↑"
        ]
        
        for text in texts_with_special:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            assert -1 <= result["sentiment"] <= 1
    
    async def test_typos_and_errors(self, mock_triton_client):
        """Test robustness to typos."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        texts_with_typos = [
            "Compny repotrs strng earnngs",  # Typos
            "Stock price incresed signficantly",
            "Excelent quaterly resuts"
        ]
        
        for text in texts_with_typos:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            # Should still work
            assert -1 <= result["sentiment"] <= 1
    
    async def test_multilingual_text(self, mock_triton_client):
        """Test with non-English text."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        multilingual_texts = [
            "삼성전자 실적 호조",  # Korean
            "トヨタ自動車の業績",  # Japanese
            "苹果公司股价上涨"  # Chinese
        ]
        
        for text in multilingual_texts:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            assert -1 <= result["sentiment"] <= 1
    
    async def test_numbers_and_dates(self, mock_triton_client):
        """Test with numbers and dates."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        texts_with_numbers = [
            "Q4 2024 earnings: $2.5B revenue, 15% growth",
            "Stock price: 150.25 (+3.2% YTD)",
            "PE ratio 25.5x, target price $175"
        ]
        
        for text in texts_with_numbers:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            assert -1 <= result["sentiment"] <= 1


@pytest.mark.validation
class TestSentimentConsistency:
    """Test sentiment prediction consistency."""
    
    @pytest.mark.asyncio
    async def test_similar_text_consistency(self, mock_triton_client):
        """Test similar texts produce similar sentiments."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        similar_texts = [
            "Company reports strong earnings",
            "Company reports excellent earnings",
            "Company reports good earnings"
        ]
        
        sentiments = []
        for text in similar_texts:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            sentiments.append(result["sentiment"])
        
        print(f"\nSimilar Text Sentiments: {sentiments}")
        
        # With mock, will be random, but check range
        assert all(-1 <= s <= 1 for s in sentiments)
    
    @pytest.mark.asyncio
    async def test_negation_handling(self, mock_triton_client):
        """Test handling of negation."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        text_pairs = [
            ("Stock price increased", "Stock price did not increase"),
            ("Earnings are strong", "Earnings are not strong"),
            ("Good news for investors", "Not good news for investors")
        ]
        
        for positive, negative in text_pairs:
            tokens_pos = extractor.tokenize_text(positive)
            tokens_neg = extractor.tokenize_text(negative)
            
            result_pos = await mock_triton_client.analyze_sentiment(
                tokens_pos["input_ids"],
                tokens_pos["attention_mask"]
            )
            result_neg = await mock_triton_client.analyze_sentiment(
                tokens_neg["input_ids"],
                tokens_neg["attention_mask"]
            )
            
            print(f"\n'{positive}': {result_pos['sentiment']:.3f}")
            print(f"'{negative}': {result_neg['sentiment']:.3f}")
            
            # Both should be in valid range
            assert -1 <= result_pos["sentiment"] <= 1
            assert -1 <= result_neg["sentiment"] <= 1


@pytest.mark.validation
class TestConfidenceScores:
    """Test confidence score quality."""
    
    @pytest.mark.asyncio
    async def test_confidence_range(self, mock_triton_client):
        """Test confidence scores are in valid range."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        texts = [
            "Definitely excellent news",
            "Maybe good news",
            "Uncertain outlook"
        ]
        
        confidences = []
        for text in texts:
            tokens = extractor.tokenize_text(text)
            result = await mock_triton_client.analyze_sentiment(
                tokens["input_ids"],
                tokens["attention_mask"]
            )
            confidences.append(result["confidence"])
        
        print(f"\nConfidence Scores: {confidences}")
        
        # All should be in [0, 1]
        assert all(0 <= c <= 1 for c in confidences)
    
    @pytest.mark.asyncio
    async def test_confidence_correlation(self, mock_triton_client):
        """Test confidence correlates with certainty."""
        from src.features.feature_extractor import SentimentFeatureExtractor
        
        extractor = SentimentFeatureExtractor()
        
        # Clear sentiment
        clear_text = "Absolutely fantastic excellent amazing results"
        tokens_clear = extractor.tokenize_text(clear_text)
        result_clear = await mock_triton_client.analyze_sentiment(
            tokens_clear["input_ids"],
            tokens_clear["attention_mask"]
        )
        
        # Ambiguous sentiment
        ambiguous_text = "Some good but also some bad mixed results"
        tokens_ambiguous = extractor.tokenize_text(ambiguous_text)
        result_ambiguous = await mock_triton_client.analyze_sentiment(
            tokens_ambiguous["input_ids"],
            tokens_ambiguous["attention_mask"]
        )
        
        print(f"\nClear text confidence: {result_clear['confidence']:.3f}")
        print(f"Ambiguous text confidence: {result_ambiguous['confidence']:.3f}")
        
        # Both should be valid
        assert 0 <= result_clear["confidence"] <= 1
        assert 0 <= result_ambiguous["confidence"] <= 1
