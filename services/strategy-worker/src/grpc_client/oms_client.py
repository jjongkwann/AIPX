"""gRPC client for Order Management Service."""

import asyncio
from typing import AsyncGenerator
from uuid import UUID

import grpc
import structlog
from tenacity import (
    retry,
    retry_if_exception_type,
    stop_after_attempt,
    wait_exponential,
)

from ..config import OMSGrpcConfig

logger = structlog.get_logger()


# Import will be added after proto generation
# from .generated import order_pb2, order_pb2_grpc


class OrderRequest:
    """Order request model."""

    def __init__(
        self,
        order_id: UUID,
        symbol: str,
        side: str,
        order_type: str,
        quantity: int,
        price: float | None = None,
        strategy_id: UUID | None = None,
    ) -> None:
        """Initialize order request.

        Args:
            order_id: Order ID
            symbol: Trading symbol
            side: Order side (BUY/SELL)
            order_type: Order type (LIMIT/MARKET)
            quantity: Order quantity
            price: Order price (required for LIMIT orders)
            strategy_id: Strategy ID (optional)
        """
        self.order_id = order_id
        self.symbol = symbol
        self.side = side
        self.order_type = order_type
        self.quantity = quantity
        self.price = price
        self.strategy_id = strategy_id


class OrderResponse:
    """Order response model."""

    def __init__(
        self,
        order_id: str,
        request_id: str,
        status: str,
        filled_price: float,
        filled_quantity: int,
        message: str,
        timestamp: int,
    ) -> None:
        """Initialize order response.

        Args:
            order_id: OMS order ID
            request_id: Original request ID
            status: Order status
            filled_price: Average filled price
            filled_quantity: Filled quantity
            message: Status message or error
            timestamp: Response timestamp
        """
        self.order_id = order_id
        self.request_id = request_id
        self.status = status
        self.filled_price = filled_price
        self.filled_quantity = filled_quantity
        self.message = message
        self.timestamp = timestamp


class OMSClient:
    """gRPC client for Order Management Service."""

    def __init__(self, config: OMSGrpcConfig) -> None:
        """Initialize OMS client.

        Args:
            config: OMS gRPC configuration
        """
        self.config = config
        self._channel: grpc.aio.Channel | None = None
        self._stub: any = None
        self._connected = False

    async def connect(self) -> None:
        """Establish gRPC connection to OMS."""
        if self._connected:
            logger.warning("OMS client already connected")
            return

        try:
            # Create gRPC channel
            options = [
                ("grpc.max_send_message_length", self.config.max_message_size),
                ("grpc.max_receive_message_length", self.config.max_message_size),
                ("grpc.keepalive_time_ms", 30000),
                ("grpc.keepalive_timeout_ms", 10000),
            ]

            self._channel = grpc.aio.insecure_channel(
                f"{self.config.host}:{self.config.port}",
                options=options,
            )

            # Wait for channel to be ready
            await asyncio.wait_for(
                self._channel.channel_ready(),
                timeout=self.config.timeout,
            )

            # Create stub (will be uncommented after proto generation)
            # self._stub = order_pb2_grpc.OrderServiceStub(self._channel)

            self._connected = True
            logger.info(
                "OMS gRPC client connected",
                host=self.config.host,
                port=self.config.port,
            )

        except asyncio.TimeoutError:
            logger.error("OMS connection timeout", host=self.config.host, port=self.config.port)
            raise
        except Exception as e:
            logger.error("Failed to connect to OMS", error=str(e))
            raise

    async def close(self) -> None:
        """Close gRPC connection."""
        if not self._connected or self._channel is None:
            return

        try:
            await self._channel.close()
            logger.info("OMS gRPC client closed")
        except Exception as e:
            logger.error("Error closing OMS client", error=str(e))
        finally:
            self._connected = False
            self._channel = None
            self._stub = None

    @retry(
        retry=retry_if_exception_type((grpc.aio.AioRpcError, asyncio.TimeoutError)),
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=10),
        reraise=True,
    )
    async def submit_order(self, order: OrderRequest) -> OrderResponse:
        """Submit single order to OMS.

        Args:
            order: Order request

        Returns:
            Order response

        Raises:
            grpc.aio.AioRpcError: If gRPC call fails
            RuntimeError: If client not connected
        """
        if not self._connected or self._stub is None:
            raise RuntimeError("OMS client not connected. Call connect() first.")

        logger.info(
            "Submitting order to OMS",
            order_id=str(order.order_id),
            symbol=order.symbol,
            side=order.side,
            quantity=order.quantity,
        )

        # This will be implemented after proto generation
        # For now, return a placeholder
        raise NotImplementedError("Will be implemented after proto generation")

        # Implementation example (to be uncommented):
        # try:
        #     request = order_pb2.OrderRequest(
        #         id=str(order.order_id),
        #         symbol=order.symbol,
        #         side=order_pb2.BUY if order.side == "BUY" else order_pb2.SELL,
        #         type=order_pb2.LIMIT if order.order_type == "LIMIT" else order_pb2.MARKET,
        #         price=order.price or 0.0,
        #         quantity=order.quantity,
        #         strategy_id=str(order.strategy_id) if order.strategy_id else "",
        #         timestamp=int(asyncio.get_event_loop().time() * 1000),
        #     )
        #
        #     response = await self._stub.SubmitOrder(request, timeout=self.config.timeout)
        #
        #     return OrderResponse(
        #         order_id=response.order_id,
        #         request_id=response.request_id,
        #         status=self._map_status(response.status),
        #         filled_price=response.filled_price,
        #         filled_quantity=response.filled_quantity,
        #         message=response.message,
        #         timestamp=response.timestamp,
        #     )
        # except grpc.aio.AioRpcError as e:
        #     logger.error(
        #         "gRPC error submitting order",
        #         order_id=str(order.order_id),
        #         error=e.details(),
        #         code=e.code(),
        #     )
        #     raise

    async def stream_orders(self, orders: AsyncGenerator[OrderRequest, None]) -> AsyncGenerator[OrderResponse, None]:
        """Stream orders to OMS and receive responses.

        Args:
            orders: Async generator of order requests

        Yields:
            Order responses

        Raises:
            RuntimeError: If client not connected
        """
        if not self._connected or self._stub is None:
            raise RuntimeError("OMS client not connected. Call connect() first.")

        logger.info("Starting order stream to OMS")

        # This will be implemented after proto generation
        raise NotImplementedError("Will be implemented after proto generation")

        # Implementation example (to be uncommented):
        # async def request_generator():
        #     async for order in orders:
        #         yield order_pb2.OrderRequest(
        #             id=str(order.order_id),
        #             symbol=order.symbol,
        #             side=order_pb2.BUY if order.side == "BUY" else order_pb2.SELL,
        #             type=order_pb2.LIMIT if order.order_type == "LIMIT" else order_pb2.MARKET,
        #             price=order.price or 0.0,
        #             quantity=order.quantity,
        #             strategy_id=str(order.strategy_id) if order.strategy_id else "",
        #             timestamp=int(asyncio.get_event_loop().time() * 1000),
        #         )
        #
        # try:
        #     async for response in self._stub.StreamOrders(request_generator()):
        #         yield OrderResponse(
        #             order_id=response.order_id,
        #             request_id=response.request_id,
        #             status=self._map_status(response.status),
        #             filled_price=response.filled_price,
        #             filled_quantity=response.filled_quantity,
        #             message=response.message,
        #             timestamp=response.timestamp,
        #         )
        # except grpc.aio.AioRpcError as e:
        #     logger.error("gRPC error in order stream", error=e.details(), code=e.code())
        #     raise

    @staticmethod
    def _map_status(proto_status: int) -> str:
        """Map protobuf status to string.

        Args:
            proto_status: Protobuf status enum value

        Returns:
            Status string
        """
        status_map = {
            0: "PENDING",
            1: "ACCEPTED",
            2: "REJECTED",
            3: "PARTIALLY_FILLED",
            4: "FILLED",
            5: "CANCELLED",
        }
        return status_map.get(proto_status, "UNKNOWN")
