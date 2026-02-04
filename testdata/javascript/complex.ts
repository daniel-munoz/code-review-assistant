/**
 * Complex TypeScript module with high cyclomatic complexity for testing.
 */

interface DataItem {
  type?: string;
  value?: unknown;
  score?: number;
  priority?: string;
  data?: Record<string, unknown>;
  values?: number[];
}

interface ProcessResult {
  processed: DataItem[];
  errors: Array<{ item: DataItem; error: string }>;
  stats: { total: number; filtered: number; errors: number };
}

// Function with high cyclomatic complexity
function highlyComplexFunction(
  data: DataItem[],
  mode: string,
  threshold: number,
  enableFilter: boolean = true,
  maxDepth: number = 5,
  callback?: (item: DataItem) => void
): ProcessResult {
  const result: ProcessResult = {
    processed: [],
    errors: [],
    stats: { total: 0, filtered: 0, errors: 0 },
  };

  if (!data || data.length === 0) {
    return result;
  }

  for (const item of data) {
    try {
      if (mode === 'strict') {
        if (!('value' in item)) {
          result.errors.push({ item, error: 'missing value' });
          result.stats.errors++;
          continue;
        } else if (item.value === null) {
          result.errors.push({ item, error: 'null value' });
          result.stats.errors++;
          continue;
        }
      }

      if (enableFilter) {
        if ('score' in item && item.score !== undefined && item.score < threshold) {
          result.stats.filtered++;
          continue;
        } else if ('priority' in item && item.priority === 'low') {
          result.stats.filtered++;
          continue;
        }
      }

      // Process based on item type
      let processed: DataItem;
      if ('type' in item && item.type) {
        const itemType = item.type;
        if (itemType === 'A') {
          processed = processTypeA(item, maxDepth);
        } else if (itemType === 'B') {
          processed = processTypeB(item, maxDepth);
        } else if (itemType === 'C' || itemType === 'D') {
          processed = processTypeCD(item, maxDepth);
        } else {
          processed = processDefault(item);
        }
      } else {
        processed = processDefault(item);
      }

      if (callback) {
        callback(processed);
      }

      result.processed.push(processed);
      result.stats.total++;
    } catch (e) {
      const error = e instanceof Error ? e.message : 'unexpected error';
      result.errors.push({ item, error });
      result.stats.errors++;
    }
  }

  return result;
}

function processTypeA(item: DataItem, depth: number): DataItem {
  return { type: 'A', data: item.data || {}, value: depth };
}

function processTypeB(item: DataItem, depth: number): DataItem {
  return { type: 'B', data: item.data || {}, value: depth };
}

function processTypeCD(item: DataItem, depth: number): DataItem {
  return { type: 'CD', data: item.data || {}, value: depth };
}

function processDefault(item: DataItem): DataItem {
  return { type: 'default', data: item };
}

// Function with deep nesting
function deeplyNestedFunction(data: number[]): number {
  let total = 0;

  for (const i of data) {
    if (i > 0) {
      for (let j = 0; j < i; j++) {
        if (j % 2 === 0) {
          for (let k = 0; k < j; k++) {
            if (k > 0) {
              if (k % 3 === 0) {
                total += k;
              }
            }
          }
        }
      }
    }
  }

  return total;
}

// Function with many return statements
function manyReturns(value: number): string {
  if (value < 0) {
    return 'negative';
  }

  if (value === 0) {
    return 'zero';
  }

  if (value < 10) {
    return 'single digit';
  }

  if (value < 100) {
    return 'double digit';
  }

  if (value < 1000) {
    return 'triple digit';
  }

  return 'large';
}

// Function with magic numbers
function usesMagicNumbers(items: number[]): Record<string, number> {
  const result: Record<string, number> = {
    small: 0,
    medium: 0,
    large: 0,
  };

  for (const item of items) {
    if (item < 10) {
      result.small++;
    } else if (item < 100) {
      result.medium++;
    } else if (item < 1000) {
      result.large++;
    }
  }

  // More magic numbers
  const timeout = 30;
  const retries = 3;
  const bufferSize = 4096;

  return result;
}

// Complex class
class ComplexClass {
  private config: Record<string, unknown>;
  private cache: Map<string, unknown>;
  private stats: Map<string, number>;

  constructor(config: Record<string, unknown>) {
    this.config = config;
    this.cache = new Map();
    this.stats = new Map();
  }

  complexMethod(
    data: unknown[],
    transform: boolean = true,
    validate: boolean = true
  ): unknown[] {
    const results: unknown[] = [];

    for (const item of data) {
      if (validate) {
        if (!this.validate(item)) {
          this.incrementStat('invalid');
          continue;
        }
      }

      let processedItem = item;
      if (transform) {
        processedItem = this.transform(item);
      }

      if (processedItem !== null) {
        results.push(processedItem);
        this.incrementStat('processed');
      }
    }

    return results;
  }

  private validate(item: unknown): boolean {
    if (item === null || item === undefined) {
      return false;
    }
    if (typeof item === 'object' && !('id' in item)) {
      return false;
    }
    return true;
  }

  private transform(item: unknown): unknown {
    if (typeof item === 'object' && item !== null) {
      return Object.fromEntries(
        Object.entries(item).filter(([_, v]) => v !== null)
      );
    }
    return item;
  }

  private incrementStat(key: string): void {
    const current = this.stats.get(key) || 0;
    this.stats.set(key, current + 1);
  }
}

// Function with ternary expressions adding complexity
const ternaryComplexity = (a: number, b: number, c: number): string => {
  return a > 0
    ? b > 0
      ? c > 0
        ? 'all positive'
        : 'c not positive'
      : c > 0
        ? 'b not positive'
        : 'b and c not positive'
    : 'a not positive';
};

// Function with logical operators adding complexity
function logicalComplexity(a: boolean, b: boolean, c: boolean, d: boolean): boolean {
  if ((a && b) || (c && d)) {
    return true;
  }
  if (a || b || c) {
    return false;
  }
  return a && b && c && d;
}

export {
  highlyComplexFunction,
  deeplyNestedFunction,
  manyReturns,
  usesMagicNumbers,
  ComplexClass,
  ternaryComplexity,
  logicalComplexity,
};
