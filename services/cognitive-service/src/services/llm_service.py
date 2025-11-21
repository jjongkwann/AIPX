"""LLM service for interacting with Claude and GPT-4."""

import asyncio
from typing import Any

import structlog
from langchain_anthropic import ChatAnthropic
from langchain_core.language_models import BaseChatModel
from langchain_core.messages import BaseMessage, HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI

from src.config import Settings

logger = structlog.get_logger()


class LLMService:
    """
    Service for managing LLM interactions.

    Supports Claude (Anthropic) and GPT-4 (OpenAI) with automatic
    fallback and retry logic.
    """

    def __init__(self, settings: Settings):
        """Initialize LLM service with configuration."""
        self.settings = settings
        self._claude: BaseChatModel | None = None
        self._openai: BaseChatModel | None = None
        self._initialized = False

    async def initialize(self) -> None:
        """Initialize LLM clients."""
        if self._initialized:
            return

        try:
            # Initialize Claude
            if self.settings.anthropic_api_key:
                self._claude = ChatAnthropic(
                    model=self.settings.claude_model,
                    anthropic_api_key=self.settings.anthropic_api_key,
                    temperature=self.settings.llm_temperature,
                    max_tokens=self.settings.llm_max_tokens,
                    timeout=self.settings.llm_timeout,
                )
                logger.info("claude_initialized", model=self.settings.claude_model)

            # Initialize OpenAI (optional)
            if self.settings.openai_api_key:
                self._openai = ChatOpenAI(
                    model=self.settings.openai_model,
                    api_key=self.settings.openai_api_key,
                    temperature=self.settings.llm_temperature,
                    max_tokens=self.settings.llm_max_tokens,
                    timeout=self.settings.llm_timeout,
                )
                logger.info("openai_initialized", model=self.settings.openai_model)

            self._initialized = True
            logger.info("llm_service_initialized")

        except Exception as e:
            logger.error("llm_initialization_failed", error=str(e))
            raise

    async def cleanup(self) -> None:
        """Cleanup LLM resources."""
        self._claude = None
        self._openai = None
        self._initialized = False
        logger.info("llm_service_cleanup_complete")

    def get_llm(self, prefer: str | None = None) -> BaseChatModel:
        """
        Get LLM instance.

        Args:
            prefer: Preferred LLM ("claude" or "openai")

        Returns:
            LLM instance

        Raises:
            RuntimeError: If no LLM is available
        """
        if not self._initialized:
            raise RuntimeError("LLM service not initialized. Call initialize() first.")

        # Use preference if specified
        if prefer == "openai" and self._openai:
            return self._openai
        if prefer == "claude" and self._claude:
            return self._claude

        # Use default
        if self.settings.default_llm == "claude" and self._claude:
            return self._claude
        if self.settings.default_llm == "openai" and self._openai:
            return self._openai

        # Fallback to any available
        if self._claude:
            return self._claude
        if self._openai:
            return self._openai

        raise RuntimeError("No LLM available")

    async def invoke(
        self,
        messages: list[BaseMessage],
        prefer: str | None = None,
        **kwargs: Any,
    ) -> BaseMessage:
        """
        Invoke LLM with messages.

        Args:
            messages: List of messages
            prefer: Preferred LLM
            **kwargs: Additional arguments for LLM

        Returns:
            LLM response message
        """
        llm = self.get_llm(prefer)

        try:
            response = await llm.ainvoke(messages, **kwargs)
            logger.debug(
                "llm_invoked",
                model=llm.model_name if hasattr(llm, "model_name") else "unknown",
                message_count=len(messages),
            )
            return response

        except Exception as e:
            logger.error(
                "llm_invocation_failed",
                error=str(e),
                model=llm.model_name if hasattr(llm, "model_name") else "unknown",
            )
            raise

    async def invoke_with_retry(
        self,
        messages: list[BaseMessage],
        max_retries: int = 3,
        prefer: str | None = None,
        **kwargs: Any,
    ) -> BaseMessage:
        """
        Invoke LLM with retry logic.

        Args:
            messages: List of messages
            max_retries: Maximum retry attempts
            prefer: Preferred LLM
            **kwargs: Additional arguments for LLM

        Returns:
            LLM response message

        Raises:
            Exception: If all retries fail
        """
        last_error = None

        for attempt in range(max_retries):
            try:
                return await self.invoke(messages, prefer, **kwargs)

            except Exception as e:
                last_error = e
                logger.warning(
                    "llm_retry",
                    attempt=attempt + 1,
                    max_retries=max_retries,
                    error=str(e),
                )

                if attempt < max_retries - 1:
                    await asyncio.sleep(2**attempt)  # Exponential backoff

        logger.error("llm_retries_exhausted", max_retries=max_retries)
        raise last_error or RuntimeError("LLM invocation failed")

    async def invoke_structured(
        self,
        messages: list[BaseMessage],
        schema: type,
        prefer: str | None = None,
        **kwargs: Any,
    ) -> Any:
        """
        Invoke LLM with structured output.

        Args:
            messages: List of messages
            schema: Pydantic schema for structured output
            prefer: Preferred LLM
            **kwargs: Additional arguments for LLM

        Returns:
            Parsed structured output
        """
        llm = self.get_llm(prefer)

        try:
            # Use LangChain's structured output feature
            structured_llm = llm.with_structured_output(schema)
            response = await structured_llm.ainvoke(messages, **kwargs)

            logger.debug(
                "llm_structured_invoked",
                model=llm.model_name if hasattr(llm, "model_name") else "unknown",
                schema=schema.__name__,
            )
            return response

        except Exception as e:
            logger.error(
                "llm_structured_invocation_failed",
                error=str(e),
                schema=schema.__name__,
            )
            raise

    def create_message(self, role: str, content: str) -> BaseMessage:
        """
        Create a message for LLM.

        Args:
            role: Message role ("system", "user", "assistant")
            content: Message content

        Returns:
            BaseMessage instance
        """
        if role == "system":
            return SystemMessage(content=content)
        return HumanMessage(content=content)
