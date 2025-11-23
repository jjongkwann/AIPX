"""Event loop for backtesting engine"""

import heapq
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from typing import Any, Callable, Dict, List


class EventType(Enum):
    """Types of events in the backtesting engine"""

    MARKET_DATA = "market_data"
    ORDER = "order"
    FILL = "fill"
    SIGNAL = "signal"


@dataclass(order=True)
class Event:
    """Event in the backtesting system"""

    timestamp: datetime = field(compare=True)
    event_type: EventType = field(compare=False)
    data: Dict[str, Any] = field(compare=False, default_factory=dict)

    def __post_init__(self):
        if not isinstance(self.timestamp, datetime):
            raise TypeError("timestamp must be a datetime object")
        if not isinstance(self.event_type, EventType):
            raise TypeError("event_type must be an EventType enum")


class EventLoop:
    """Event-driven loop for backtesting"""

    def __init__(self):
        self._event_queue: List[Event] = []
        self._handlers: Dict[EventType, List[Callable]] = {event_type: [] for event_type in EventType}
        self._is_running = False

    def register_handler(self, event_type: EventType, handler: Callable):
        """Register an event handler for a specific event type"""
        if not isinstance(event_type, EventType):
            raise TypeError("event_type must be an EventType enum")
        if not callable(handler):
            raise TypeError("handler must be callable")
        self._handlers[event_type].append(handler)

    def add_event(self, event: Event):
        """Add an event to the queue (maintains priority by timestamp)"""
        if not isinstance(event, Event):
            raise TypeError("event must be an Event instance")
        heapq.heappush(self._event_queue, event)

    def process_events(self) -> int:
        """Process all events in the queue and return count of processed events"""
        self._is_running = True
        processed_count = 0

        while self._event_queue and self._is_running:
            event = heapq.heappop(self._event_queue)
            self._process_event(event)
            processed_count += 1

        return processed_count

    def _process_event(self, event: Event):
        """Process a single event by calling all registered handlers"""
        handlers = self._handlers.get(event.event_type, [])

        for handler in handlers:
            try:
                handler(event)
            except Exception as e:
                # Log error but continue processing
                print(f"Error in handler for {event.event_type}: {e}")
                raise  # Re-raise for testing purposes

    def stop(self):
        """Stop the event loop"""
        self._is_running = False

    def clear(self):
        """Clear all events from the queue"""
        self._event_queue.clear()

    @property
    def queue_size(self) -> int:
        """Return the current size of the event queue"""
        return len(self._event_queue)

    @property
    def is_running(self) -> bool:
        """Check if the event loop is running"""
        return self._is_running
