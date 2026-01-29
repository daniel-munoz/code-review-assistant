"""Complex Python module with high cyclomatic complexity for testing."""

from typing import Any, Dict, List, Optional, Union


def highly_complex_function(
    data: List[Dict[str, Any]],
    mode: str,
    threshold: float,
    enable_filter: bool = True,
    max_depth: int = 5,
    callback: Optional[callable] = None,
) -> Dict[str, Any]:
    """
    A function with high cyclomatic complexity.

    This function has many branches and conditions to test
    complexity calculation.
    """
    result = {
        "processed": [],
        "errors": [],
        "stats": {"total": 0, "filtered": 0, "errors": 0},
    }

    if not data:
        return result

    for item in data:
        try:
            if mode == "strict":
                if "value" not in item:
                    result["errors"].append({"item": item, "error": "missing value"})
                    result["stats"]["errors"] += 1
                    continue
                elif item["value"] is None:
                    result["errors"].append({"item": item, "error": "null value"})
                    result["stats"]["errors"] += 1
                    continue

            if enable_filter:
                if "score" in item and item["score"] < threshold:
                    result["stats"]["filtered"] += 1
                    continue
                elif "priority" in item and item["priority"] == "low":
                    result["stats"]["filtered"] += 1
                    continue

            # Process based on item type
            if "type" in item:
                item_type = item["type"]
                if item_type == "A":
                    processed = _process_type_a(item, max_depth)
                elif item_type == "B":
                    processed = _process_type_b(item, max_depth)
                elif item_type == "C" or item_type == "D":
                    processed = _process_type_cd(item, max_depth)
                else:
                    processed = _process_default(item)
            else:
                processed = _process_default(item)

            if callback is not None:
                callback(processed)

            result["processed"].append(processed)
            result["stats"]["total"] += 1

        except KeyError as e:
            result["errors"].append({"item": item, "error": f"key error: {e}"})
            result["stats"]["errors"] += 1
        except ValueError as e:
            result["errors"].append({"item": item, "error": f"value error: {e}"})
            result["stats"]["errors"] += 1
        except Exception as e:
            result["errors"].append({"item": item, "error": f"unexpected: {e}"})
            result["stats"]["errors"] += 1

    return result


def _process_type_a(item: Dict, depth: int) -> Dict:
    """Process type A items."""
    return {"type": "A", "data": item.get("data", {}), "depth": depth}


def _process_type_b(item: Dict, depth: int) -> Dict:
    """Process type B items."""
    return {"type": "B", "data": item.get("data", {}), "depth": depth}


def _process_type_cd(item: Dict, depth: int) -> Dict:
    """Process type C or D items."""
    return {"type": "CD", "data": item.get("data", {}), "depth": depth}


def _process_default(item: Dict) -> Dict:
    """Process items with default handling."""
    return {"type": "default", "data": item}


def deeply_nested_function(data: List[int]) -> int:
    """Function with deep nesting to test nesting depth detection."""
    total = 0

    for i in data:
        if i > 0:
            for j in range(i):
                if j % 2 == 0:
                    for k in range(j):
                        if k > 0:
                            if k % 3 == 0:
                                total += k

    return total


def many_returns(value: int) -> str:
    """Function with many return statements."""
    if value < 0:
        return "negative"

    if value == 0:
        return "zero"

    if value < 10:
        return "single digit"

    if value < 100:
        return "double digit"

    if value < 1000:
        return "triple digit"

    return "large"


def uses_magic_numbers(items: List[int]) -> Dict[str, int]:
    """Function that uses magic numbers (anti-pattern)."""
    result = {
        "small": 0,
        "medium": 0,
        "large": 0,
    }

    for item in items:
        if item < 10:
            result["small"] += 1
        elif item < 100:
            result["medium"] += 1
        elif item < 1000:
            result["large"] += 1

    # More magic numbers
    timeout = 30
    retries = 3
    buffer_size = 4096

    return result


# List comprehension complexity
def comprehension_complexity(data: List[Dict]) -> List[int]:
    """Uses comprehensions which add to complexity."""
    # Each comprehension adds complexity
    values = [item["value"] for item in data if "value" in item]
    filtered = [v for v in values if v > 0 and v < 100]
    squared = {v: v * v for v in filtered if v % 2 == 0}
    unique = {v for v in filtered}

    return list(squared.values())


class ComplexClass:
    """A class with complex methods."""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.cache = {}
        self.stats = defaultdict(int)

    def complex_method(
        self,
        data: List[Any],
        transform: bool = True,
        validate: bool = True,
    ) -> List[Any]:
        """Method with multiple parameters and high complexity."""
        results = []

        for item in data:
            if validate:
                if not self._validate(item):
                    self.stats["invalid"] += 1
                    continue

            if transform:
                item = self._transform(item)

            if item is not None:
                results.append(item)
                self.stats["processed"] += 1

        return results

    def _validate(self, item: Any) -> bool:
        """Validate an item."""
        if item is None:
            return False
        if isinstance(item, dict) and "id" not in item:
            return False
        return True

    def _transform(self, item: Any) -> Any:
        """Transform an item."""
        if isinstance(item, dict):
            return {k: v for k, v in item.items() if v is not None}
        return item
