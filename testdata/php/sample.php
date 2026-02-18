<?php
/**
 * Sample PHP module for testing the code analyzer.
 *
 * This module contains various functions and classes to test
 * metrics extraction and anti-pattern detection.
 */

namespace App\Services;

use App\Models\User;
use App\Contracts\UserRepositoryInterface;
use RuntimeException;
use DateTime;

// Simple function
function greet(string $name): string
{
    return "Hello, {$name}!";
}

// Function with multiple parameters
function calculateTotal(array $items, float $taxRate, float $discount = 0.0): float
{
    $subtotal = array_sum($items);
    $tax = $subtotal * $taxRate;
    $total = $subtotal + $tax - $discount;
    return $total;
}

// Function with complexity
function processData(array $data, ?string $filterKey = null): array
{
    $result = [];

    foreach ($data as $item) {
        if ($filterKey === null) {
            $result[] = $item;
        } elseif (isset($item[$filterKey])) {
            if ($item[$filterKey] !== null) {
                $result[] = $item;
            }
        }
    }

    return $result;
}

class DataProcessor
{
    private string $name;
    private int $maxItems;
    private array $items = [];

    public function __construct(string $name, int $maxItems = 100)
    {
        $this->name = $name;
        $this->maxItems = $maxItems;
    }

    public function addItem(array $item): bool
    {
        if (count($this->items) < $this->maxItems) {
            $this->items[] = $item;
            return true;
        }
        return false;
    }

    public function getItems(): array
    {
        return $this->items;
    }

    public function clear(): void
    {
        $this->items = [];
    }
}

class AdvancedProcessor extends DataProcessor
{
    private bool $enableLogging;
    private array $log = [];

    public function __construct(string $name, int $maxItems = 100, bool $enableLogging = false)
    {
        parent::__construct($name, $maxItems);
        $this->enableLogging = $enableLogging;
    }

    public function addItem(array $item): bool
    {
        $result = parent::addItem($item);
        if ($this->enableLogging) {
            $this->log[] = "Added item: " . ($result ? 'true' : 'false');
        }
        return $result;
    }

    public function getLog(): array
    {
        return $this->log;
    }
}

// Constants
const MAX_RETRIES = 3;
const DEFAULT_TIMEOUT = 30;
const PI = 3.14159;

function main(): void
{
    $processor = new DataProcessor('test');
    $processor->addItem(['id' => 1, 'value' => 'test']);
    echo greet('World');
}
