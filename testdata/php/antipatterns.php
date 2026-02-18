<?php
/**
 * PHP module with anti-patterns for testing detector rules.
 */

namespace App\AntiPatterns;

// Anti-pattern: Too many parameters (more than 5)
function functionWithTooManyParams(
    string $param1,
    int $param2,
    bool $param3,
    array $param4,
    array $param5,
    \DateTime $param6,
    string $param7 = 'default'
): array {
    return [
        'param1' => $param1,
        'param2' => $param2,
        'param3' => $param3,
        'param4' => $param4,
        'param5' => $param5,
        'param6' => $param6,
        'param7' => $param7,
    ];
}

// Anti-pattern: Deep nesting (more than 4 levels)
function deeplyNestedCode(array $data): int
{
    $total = 0;

    foreach ($data as $level1) {
        // Level 1
        if (count($level1) > 0) {
            // Level 2
            foreach ($level1 as $level2) {
                // Level 3
                if (count($level2) > 0) {
                    // Level 4
                    foreach ($level2 as $level3) {
                        // Level 5 - too deep!
                        if ($level3 > 0) {
                            // Level 6 - way too deep!
                            $total += $level3;
                        }
                    }
                }
            }
        }
    }

    return $total;
}

// Anti-pattern: Multiple return statements (more than 5)
function tooManyReturns(int $value, string $mode): string
{
    if ($value < 0) {
        return 'negative';
    }

    if ($mode === 'strict') {
        if ($value === 0) {
            return 'zero-strict';
        }
        return 'positive-strict';
    }

    if ($mode === 'lenient') {
        return 'lenient-mode';
    }

    if ($value === 0) {
        return 'zero';
    }

    if ($value < 10) {
        return 'small';
    }

    return 'large';
}

// Anti-pattern: Magic numbers throughout the code
function magicNumbersEverywhere(array $items): array
{
    $result = [
        'categoryA' => 0,
        'categoryB' => 0,
        'categoryC' => 0,
    ];

    foreach ($items as $item) {
        if ($item < 42) {
            $result['categoryA']++;
        } elseif ($item < 256) {
            $result['categoryB']++;
        } elseif ($item < 1024) {
            $result['categoryC']++;
        }
    }

    // More magic numbers
    $threshold = 0.75;
    $multiplier = 2.5;

    foreach ($result as $key => $val) {
        $result[$key] = (int)floor($val * $multiplier);
    }

    return $result;
}

// Class with anti-patterns
class AntiPatternClass
{
    private string $name;
    private int $value;
    private array $config;
    private bool $enabled;
    private int $timeout;
    private int $retries;
    private ?\Closure $callback;

    // Anti-pattern: Too many parameters in constructor
    public function __construct(
        string $name,
        int $value,
        array $config,
        bool $enabled,
        int $timeout,
        int $retries,
        ?\Closure $callback = null
    ) {
        $this->name = $name;
        $this->value = $value;
        $this->config = $config;
        $this->enabled = $enabled;
        $this->timeout = $timeout;
        $this->retries = $retries;
        $this->callback = $callback;
    }

    // Anti-pattern: Method with high complexity and deep nesting
    public function complexNestedMethod(array $data): int
    {
        $count = 0;

        foreach ($data as $item) {
            if (isset($item['type'])) {
                if ($item['type'] === 'A') {
                    if (isset($item['value'])) {
                        if ($item['value'] > 0) {
                            if ($item['value'] < 100) {
                                $count += $item['value'];
                            }
                        }
                    }
                } elseif ($item['type'] === 'B') {
                    if (isset($item['values']) && is_array($item['values'])) {
                        foreach ($item['values'] as $v) {
                            if ($v !== null) {
                                $count += $v;
                            }
                        }
                    }
                }
            }
        }

        return $count;
    }

    // Anti-pattern: Multiple returns in a method
    public function methodWithManyReturns(string $key): mixed
    {
        if (empty($key)) {
            return null;
        }

        if (!array_key_exists($key, $this->config)) {
            return 'not found';
        }

        $value = $this->config[$key];

        if ($value === null) {
            return 'null';
        }

        if (is_string($value)) {
            return strtoupper($value);
        }

        if (is_int($value)) {
            return $value * 2;
        }

        return $value;
    }
}

// Function combining multiple anti-patterns
function combinedAntipatterns(
    int $a,
    int $b,
    int $c,
    int $d,
    int $e,
    int $f
): int {
    $result = 0;

    // Deep nesting
    if ($a > 0) {
        if ($b > 0) {
            if ($c > 0) {
                if ($d > 0) {
                    if ($e > 0) {
                        // Too deep
                        $result = $a + $b + $c + $d + $e + $f;
                    }
                }
            }
        }
    }

    // Magic numbers
    if ($result < 42) {
        $result = 500;
    } elseif ($result < 256) {
        $result = 500;
    }

    // Multiple returns
    if ($result === 0) {
        return -1;
    }

    if ($result < 0) {
        return 0;
    }

    return $result;
}
