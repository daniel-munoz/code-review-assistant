"""Sample Python module for testing the code analyzer.

This module contains various functions and classes to test
metrics extraction and anti-pattern detection.
"""

import os
import sys
from typing import List, Optional
from collections import defaultdict


# Simple function
def greet(name: str) -> str:
    """Return a greeting message."""
    return f"Hello, {name}!"


# Function with multiple parameters
def calculate_total(items: List[float], tax_rate: float, discount: float = 0.0) -> float:
    """Calculate total with tax and discount."""
    subtotal = sum(items)
    tax = subtotal * tax_rate
    total = subtotal + tax - discount
    return total


# Function with complexity
def process_data(data: List[dict], filter_key: Optional[str] = None) -> List[dict]:
    """Process and filter data based on conditions."""
    result = []

    for item in data:
        if filter_key is None:
            result.append(item)
        elif filter_key in item:
            if item[filter_key] is not None:
                result.append(item)

    return result


class DataProcessor:
    """A class for processing data with various methods."""

    def __init__(self, name: str, max_items: int = 100):
        """Initialize the processor with a name and max items."""
        self.name = name
        self.max_items = max_items
        self.items = []

    def add_item(self, item: dict) -> bool:
        """Add an item if under the limit."""
        if len(self.items) < self.max_items:
            self.items.append(item)
            return True
        return False

    def get_items(self) -> List[dict]:
        """Return all items."""
        return self.items.copy()

    def clear(self) -> None:
        """Clear all items."""
        self.items = []


class AdvancedProcessor(DataProcessor):
    """Extended processor with additional features."""

    def __init__(self, name: str, max_items: int = 100, enable_logging: bool = False):
        """Initialize with logging option."""
        super().__init__(name, max_items)
        self.enable_logging = enable_logging
        self.log = []

    def add_item(self, item: dict) -> bool:
        """Add item with optional logging."""
        result = super().add_item(item)
        if self.enable_logging:
            self.log.append(f"Added item: {result}")
        return result

    def get_log(self) -> List[str]:
        """Return the log entries."""
        return self.log.copy()


# Constants
MAX_RETRIES = 3
DEFAULT_TIMEOUT = 30
PI = 3.14159


def main():
    """Main entry point."""
    processor = DataProcessor("test")
    processor.add_item({"id": 1, "value": "test"})
    print(greet("World"))


if __name__ == "__main__":
    main()
