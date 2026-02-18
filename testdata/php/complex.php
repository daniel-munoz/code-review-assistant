<?php
/**
 * Complex PHP module for testing complexity calculations.
 */

namespace App\Complex;

use App\Models\User;
use Psr\Log\LoggerInterface;

// Highly complex function
function highlyComplexFunction(
    array $data,
    string $mode,
    int $threshold,
    bool $strict,
    ?string $filter,
    int $maxRetries
): array {
    $results = [];
    $errors = [];
    $retryCount = 0;

    foreach ($data as $key => $item) {
        if (!is_array($item)) {
            continue;
        }

        if ($mode === 'strict') {
            if (isset($item['value'])) {
                if ($item['value'] > $threshold) {
                    $results[$key] = $item;
                } elseif ($item['value'] === $threshold && $strict) {
                    $results[$key] = $item;
                }
            }
        } elseif ($mode === 'lenient') {
            if ($filter !== null) {
                if (isset($item[$filter]) && $item[$filter] !== '') {
                    $results[$key] = $item;
                }
            } else {
                $results[$key] = $item;
            }
        } elseif ($mode === 'retry') {
            while ($retryCount < $maxRetries) {
                try {
                    if (isset($item['value']) && $item['value'] > 0) {
                        $results[$key] = $item;
                        break;
                    }
                } catch (\Exception $e) {
                    $errors[] = $e->getMessage();
                    $retryCount++;
                }
            }
        }
    }

    return ['results' => $results, 'errors' => $errors];
}

// Deeply nested function
function deeplyNestedFunction(array $matrix): int
{
    $sum = 0;

    foreach ($matrix as $row) {
        if (is_array($row)) {
            foreach ($row as $cell) {
                if (is_array($cell)) {
                    foreach ($cell as $value) {
                        if (is_numeric($value)) {
                            if ($value > 0) {
                                $sum += $value;
                            }
                        }
                    }
                }
            }
        }
    }

    return $sum;
}

class ComplexService
{
    private LoggerInterface $logger;
    private array $cache = [];

    public function __construct(LoggerInterface $logger)
    {
        $this->logger = $logger;
    }

    public function processWithRetry(array $items, int $maxRetries = 3): array
    {
        $results = [];
        $failedItems = [];

        foreach ($items as $item) {
            $success = false;
            $attempts = 0;

            while (!$success && $attempts < $maxRetries) {
                try {
                    $result = $this->processItem($item);
                    if ($result !== null) {
                        $results[] = $result;
                        $success = true;
                    }
                } catch (\RuntimeException $e) {
                    $this->logger->warning("Retry {$attempts}: " . $e->getMessage());
                    $attempts++;
                } catch (\Exception $e) {
                    $this->logger->error("Failed: " . $e->getMessage());
                    $failedItems[] = $item;
                    break;
                }
            }

            if (!$success) {
                $failedItems[] = $item;
            }
        }

        return ['results' => $results, 'failures' => $failedItems];
    }

    private function processItem(array $item): ?array
    {
        if (!isset($item['id'])) {
            return null;
        }

        if (isset($this->cache[$item['id']])) {
            return $this->cache[$item['id']];
        }

        $processed = [
            'id' => $item['id'],
            'processed' => true,
            'timestamp' => time(),
        ];

        $this->cache[$item['id']] = $processed;
        return $processed;
    }
}

interface Processable
{
    public function process(): mixed;
    public function validate(): bool;
}

trait Cacheable
{
    private array $traitCache = [];

    public function getCached(string $key): mixed
    {
        return $this->traitCache[$key] ?? null;
    }

    public function setCached(string $key, mixed $value): void
    {
        $this->traitCache[$key] = $value;
    }
}

enum Status: string
{
    case Active = 'active';
    case Inactive = 'inactive';
    case Pending = 'pending';

    public function label(): string
    {
        return match($this) {
            self::Active => 'Active',
            self::Inactive => 'Inactive',
            self::Pending => 'Pending',
        };
    }
}
