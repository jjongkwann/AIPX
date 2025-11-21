"""Unit tests for Event Loop"""

import pytest
from datetime import datetime, timedelta
from src.engine.event_loop import EventLoop, Event, EventType


class TestEvent:
    """Test Event class"""

    def test_event_creation(self):
        """Test creating a valid event"""
        timestamp = datetime(2024, 1, 1, 9, 0)
        event = Event(
            timestamp=timestamp,
            event_type=EventType.MARKET_DATA,
            data={'symbol': '005930', 'price': 71000}
        )

        assert event.timestamp == timestamp
        assert event.event_type == EventType.MARKET_DATA
        assert event.data['symbol'] == '005930'
        assert event.data['price'] == 71000

    def test_event_ordering_by_timestamp(self):
        """Test that events are ordered by timestamp"""
        event1 = Event(
            timestamp=datetime(2024, 1, 1, 9, 0),
            event_type=EventType.MARKET_DATA
        )
        event2 = Event(
            timestamp=datetime(2024, 1, 1, 9, 1),
            event_type=EventType.ORDER
        )

        assert event1 < event2
        assert not event1 > event2

    def test_event_invalid_timestamp(self):
        """Test that invalid timestamp raises error"""
        with pytest.raises(TypeError):
            Event(
                timestamp="2024-01-01",  # String instead of datetime
                event_type=EventType.MARKET_DATA
            )

    def test_event_invalid_type(self):
        """Test that invalid event type raises error"""
        with pytest.raises(TypeError):
            Event(
                timestamp=datetime(2024, 1, 1),
                event_type="market_data"  # String instead of enum
            )


class TestEventLoop:
    """Test EventLoop class"""

    def test_event_loop_initialization(self):
        """Test creating an event loop"""
        loop = EventLoop()

        assert loop.queue_size == 0
        assert not loop.is_running
        assert loop._handlers is not None

    def test_register_handler(self):
        """Test registering event handlers"""
        loop = EventLoop()
        handler_called = []

        def handler(event):
            handler_called.append(event)

        loop.register_handler(EventType.MARKET_DATA, handler)

        # Add and process event
        event = Event(
            timestamp=datetime(2024, 1, 1),
            event_type=EventType.MARKET_DATA
        )
        loop.add_event(event)
        loop.process_events()

        assert len(handler_called) == 1
        assert handler_called[0] == event

    def test_register_multiple_handlers(self):
        """Test registering multiple handlers for same event type"""
        loop = EventLoop()
        calls = []

        def handler1(event):
            calls.append('handler1')

        def handler2(event):
            calls.append('handler2')

        loop.register_handler(EventType.MARKET_DATA, handler1)
        loop.register_handler(EventType.MARKET_DATA, handler2)

        event = Event(
            timestamp=datetime(2024, 1, 1),
            event_type=EventType.MARKET_DATA
        )
        loop.add_event(event)
        loop.process_events()

        assert len(calls) == 2
        assert 'handler1' in calls
        assert 'handler2' in calls

    def test_register_handler_invalid_type(self):
        """Test that registering handler with invalid type raises error"""
        loop = EventLoop()

        with pytest.raises(TypeError):
            loop.register_handler("market_data", lambda e: None)

    def test_register_handler_not_callable(self):
        """Test that registering non-callable raises error"""
        loop = EventLoop()

        with pytest.raises(TypeError):
            loop.register_handler(EventType.MARKET_DATA, "not_callable")

    def test_add_event(self):
        """Test adding events to queue"""
        loop = EventLoop()

        event = Event(
            timestamp=datetime(2024, 1, 1),
            event_type=EventType.MARKET_DATA
        )
        loop.add_event(event)

        assert loop.queue_size == 1

    def test_add_event_invalid(self):
        """Test that adding invalid event raises error"""
        loop = EventLoop()

        with pytest.raises(TypeError):
            loop.add_event("not_an_event")

    def test_event_queue_ordering(self):
        """Test that events are processed in timestamp order"""
        loop = EventLoop()
        processed_order = []

        def handler(event):
            processed_order.append(event.timestamp)

        loop.register_handler(EventType.MARKET_DATA, handler)
        loop.register_handler(EventType.ORDER, handler)

        # Add events in random order
        event2 = Event(
            timestamp=datetime(2024, 1, 1, 9, 2),
            event_type=EventType.ORDER
        )
        event1 = Event(
            timestamp=datetime(2024, 1, 1, 9, 0),
            event_type=EventType.MARKET_DATA
        )
        event3 = Event(
            timestamp=datetime(2024, 1, 1, 9, 1),
            event_type=EventType.SIGNAL
        )

        loop.add_event(event2)
        loop.add_event(event1)
        loop.add_event(event3)

        assert loop.queue_size == 3

        loop.process_events()

        # Should be processed in timestamp order
        assert processed_order == [
            datetime(2024, 1, 1, 9, 0),
            datetime(2024, 1, 1, 9, 1),
            datetime(2024, 1, 1, 9, 2)
        ]

    def test_duplicate_timestamps(self):
        """Test handling events with duplicate timestamps"""
        loop = EventLoop()
        processed = []

        def handler(event):
            processed.append(event.event_type)

        loop.register_handler(EventType.MARKET_DATA, handler)
        loop.register_handler(EventType.ORDER, handler)

        same_time = datetime(2024, 1, 1, 9, 0)

        event1 = Event(timestamp=same_time, event_type=EventType.MARKET_DATA)
        event2 = Event(timestamp=same_time, event_type=EventType.ORDER)

        loop.add_event(event1)
        loop.add_event(event2)

        loop.process_events()

        # Both should be processed
        assert len(processed) == 2

    def test_process_events_returns_count(self):
        """Test that process_events returns number of processed events"""
        loop = EventLoop()

        for i in range(5):
            event = Event(
                timestamp=datetime(2024, 1, 1, 9, i),
                event_type=EventType.MARKET_DATA
            )
            loop.add_event(event)

        count = loop.process_events()
        assert count == 5

    def test_empty_queue_processing(self):
        """Test processing empty queue"""
        loop = EventLoop()
        count = loop.process_events()

        assert count == 0
        assert loop.queue_size == 0

    def test_handler_error_handling(self):
        """Test that handler errors are caught and re-raised"""
        loop = EventLoop()

        def faulty_handler(event):
            raise ValueError("Handler error")

        loop.register_handler(EventType.MARKET_DATA, faulty_handler)

        event = Event(
            timestamp=datetime(2024, 1, 1),
            event_type=EventType.MARKET_DATA
        )
        loop.add_event(event)

        # Should raise the error
        with pytest.raises(ValueError, match="Handler error"):
            loop.process_events()

    def test_stop_event_loop(self):
        """Test stopping event loop"""
        loop = EventLoop()

        # Add many events
        for i in range(10):
            event = Event(
                timestamp=datetime(2024, 1, 1, 9, i),
                event_type=EventType.MARKET_DATA
            )
            loop.add_event(event)

        # Stop after processing first event
        def handler(event):
            loop.stop()

        loop.register_handler(EventType.MARKET_DATA, handler)

        count = loop.process_events()

        # Should only process one event
        assert count == 1
        assert loop.queue_size == 9  # 9 events remaining

    def test_clear_queue(self):
        """Test clearing event queue"""
        loop = EventLoop()

        for i in range(5):
            event = Event(
                timestamp=datetime(2024, 1, 1, 9, i),
                event_type=EventType.MARKET_DATA
            )
            loop.add_event(event)

        assert loop.queue_size == 5

        loop.clear()

        assert loop.queue_size == 0

    def test_multiple_event_types(self):
        """Test processing different event types"""
        loop = EventLoop()
        market_data_count = []
        order_count = []

        def market_handler(event):
            market_data_count.append(1)

        def order_handler(event):
            order_count.append(1)

        loop.register_handler(EventType.MARKET_DATA, market_handler)
        loop.register_handler(EventType.ORDER, order_handler)

        # Add various event types
        loop.add_event(Event(
            timestamp=datetime(2024, 1, 1, 9, 0),
            event_type=EventType.MARKET_DATA
        ))
        loop.add_event(Event(
            timestamp=datetime(2024, 1, 1, 9, 1),
            event_type=EventType.ORDER
        ))
        loop.add_event(Event(
            timestamp=datetime(2024, 1, 1, 9, 2),
            event_type=EventType.MARKET_DATA
        ))

        loop.process_events()

        assert len(market_data_count) == 2
        assert len(order_count) == 1
