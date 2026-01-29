"""Python module with anti-patterns for testing detector rules."""

from typing import List, Dict, Any


# Anti-pattern: Too many parameters (more than 5)
def function_with_too_many_params(
    param1: str,
    param2: int,
    param3: float,
    param4: bool,
    param5: List[str],
    param6: Dict[str, Any],
    param7: str = "default",
) -> Dict[str, Any]:
    """Function with too many parameters - should be flagged."""
    return {
        "param1": param1,
        "param2": param2,
        "param3": param3,
        "param4": param4,
        "param5": param5,
        "param6": param6,
        "param7": param7,
    }


# Anti-pattern: Deep nesting (more than 4 levels)
def deeply_nested_code(data: List[List[List[int]]]) -> int:
    """Function with deeply nested code - should be flagged."""
    total = 0

    for level1 in data:  # Level 1
        if level1:  # Level 2
            for level2 in level1:  # Level 3
                if level2:  # Level 4
                    for level3 in level2:  # Level 5 - too deep!
                        if level3 > 0:  # Level 6 - way too deep!
                            total += level3

    return total


# Anti-pattern: Multiple return statements (more than 3)
def too_many_returns(value: int, mode: str) -> str:
    """Function with too many return statements - should be flagged."""
    if value < 0:
        return "negative"

    if mode == "strict":
        if value == 0:
            return "zero-strict"
        return "positive-strict"

    if mode == "lenient":
        return "lenient-mode"

    if value == 0:
        return "zero"

    if value < 10:
        return "small"

    return "large"


# Anti-pattern: Magic numbers throughout the code
def magic_numbers_everywhere(items: List[int]) -> Dict[str, int]:
    """Function using magic numbers instead of named constants."""
    result = {"category_a": 0, "category_b": 0, "category_c": 0}

    for item in items:
        if item < 42:  # Magic number
            result["category_a"] += 1
        elif item < 256:  # Magic number
            result["category_b"] += 1
        elif item < 1024:  # Magic number
            result["category_c"] += 1

    # More magic numbers
    threshold = 0.75
    multiplier = 2.5
    offset = 100

    for key in result:
        result[key] = int(result[key] * multiplier + offset)

    return result


# Anti-pattern: Bare except (catches everything including KeyboardInterrupt)
def bare_except_handler(data: Dict[str, Any]) -> Any:
    """Function with bare except clause - Python-specific anti-pattern."""
    try:
        return data["key"]["nested"]["value"]
    except:  # noqa: E722 - Bare except is intentional for testing
        return None


# Anti-pattern: Mutable default argument
def mutable_default_arg(items: List[str] = []) -> List[str]:  # noqa: B006
    """Function with mutable default argument - Python-specific anti-pattern."""
    items.append("new_item")
    return items


class AntiPatternClass:
    """Class demonstrating various anti-patterns."""

    # Anti-pattern: Too many parameters in __init__
    def __init__(
        self,
        name: str,
        value: int,
        config: Dict,
        enabled: bool,
        timeout: int,
        retries: int,
        callback: Any = None,
    ):
        """Constructor with too many parameters."""
        self.name = name
        self.value = value
        self.config = config
        self.enabled = enabled
        self.timeout = timeout
        self.retries = retries
        self.callback = callback

    # Anti-pattern: Method with high complexity and deep nesting
    def complex_nested_method(self, data: List[Dict]) -> int:
        """Method with both high complexity and deep nesting."""
        count = 0

        for item in data:
            if "type" in item:
                if item["type"] == "A":
                    if "value" in item:
                        if item["value"] > 0:
                            if item["value"] < 100:
                                count += item["value"]
                elif item["type"] == "B":
                    if "values" in item:
                        for v in item["values"]:
                            if v is not None:
                                count += v

        return count

    # Anti-pattern: Multiple returns in a method
    def method_with_many_returns(self, key: str) -> Any:
        """Method with too many return points."""
        if not key:
            return None

        if key not in self.config:
            return "not found"

        value = self.config[key]

        if value is None:
            return "null"

        if isinstance(value, str):
            return value.upper()

        if isinstance(value, int):
            return value * 2

        return value


# Function that combines multiple anti-patterns
def combined_antipatterns(
    a: int, b: int, c: int, d: int, e: int, f: int  # Too many params
) -> int:
    """Function combining multiple anti-patterns for comprehensive testing."""
    result = 0

    # Deep nesting
    if a > 0:
        if b > 0:
            if c > 0:
                if d > 0:
                    if e > 0:  # Too deep
                        result = a + b + c + d + e + f

    # Magic numbers
    if result < 42:
        result = 100
    elif result < 256:
        result = 500

    # Multiple returns embedded
    if result == 0:
        return -1

    if result < 0:
        return 0

    return result
