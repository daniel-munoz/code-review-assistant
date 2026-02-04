/**
 * TypeScript module with anti-patterns for testing detector rules.
 */

// Anti-pattern: Too many parameters (more than 5)
function functionWithTooManyParams(
  param1: string,
  param2: number,
  param3: boolean,
  param4: string[],
  param5: Record<string, unknown>,
  param6: Date,
  param7: string = 'default'
): Record<string, unknown> {
  return {
    param1,
    param2,
    param3,
    param4,
    param5,
    param6,
    param7,
  };
}

// Anti-pattern: Deep nesting (more than 4 levels)
function deeplyNestedCode(data: number[][][]): number {
  let total = 0;

  for (const level1 of data) {
    // Level 1
    if (level1.length > 0) {
      // Level 2
      for (const level2 of level1) {
        // Level 3
        if (level2.length > 0) {
          // Level 4
          for (const level3 of level2) {
            // Level 5 - too deep!
            if (level3 > 0) {
              // Level 6 - way too deep!
              total += level3;
            }
          }
        }
      }
    }
  }

  return total;
}

// Anti-pattern: Multiple return statements (more than 5)
function tooManyReturns(value: number, mode: string): string {
  if (value < 0) {
    return 'negative';
  }

  if (mode === 'strict') {
    if (value === 0) {
      return 'zero-strict';
    }
    return 'positive-strict';
  }

  if (mode === 'lenient') {
    return 'lenient-mode';
  }

  if (value === 0) {
    return 'zero';
  }

  if (value < 10) {
    return 'small';
  }

  return 'large';
}

// Anti-pattern: Magic numbers throughout the code
function magicNumbersEverywhere(items: number[]): Record<string, number> {
  const result: Record<string, number> = {
    categoryA: 0,
    categoryB: 0,
    categoryC: 0,
  };

  for (const item of items) {
    if (item < 42) {
      // Magic number
      result.categoryA++;
    } else if (item < 256) {
      // Magic number
      result.categoryB++;
    } else if (item < 1024) {
      // Magic number
      result.categoryC++;
    }
  }

  // More magic numbers
  const threshold = 0.75;
  const multiplier = 2.5;
  const offset = 100;

  for (const key of Object.keys(result)) {
    result[key] = Math.floor(result[key] * multiplier + offset);
  }

  return result;
}

// Anti-pattern: Function with magic numbers specific to plan
function calculatePrice(basePrice: number, quantity: number): number {
  // Magic numbers: 29.99, 0.0875, 7.99
  const shippingCost = 7.99;
  const taxRate = 0.0875;
  const minPrice = 29.99;

  let total = basePrice * quantity;

  if (total < minPrice) {
    total = minPrice;
  }

  total += total * taxRate;
  total += shippingCost;

  return total;
}

// Class with anti-patterns
class AntiPatternClass {
  private name: string;
  private value: number;
  private config: Record<string, unknown>;
  private enabled: boolean;
  private timeout: number;
  private retries: number;
  private callback: (() => void) | null;

  // Anti-pattern: Too many parameters in constructor
  constructor(
    name: string,
    value: number,
    config: Record<string, unknown>,
    enabled: boolean,
    timeout: number,
    retries: number,
    callback: (() => void) | null = null
  ) {
    this.name = name;
    this.value = value;
    this.config = config;
    this.enabled = enabled;
    this.timeout = timeout;
    this.retries = retries;
    this.callback = callback;
  }

  // Anti-pattern: Method with high complexity and deep nesting
  complexNestedMethod(data: Array<{ type?: string; value?: number; values?: number[] }>): number {
    let count = 0;

    for (const item of data) {
      if ('type' in item) {
        if (item.type === 'A') {
          if ('value' in item) {
            if (item.value !== undefined && item.value > 0) {
              if (item.value < 100) {
                count += item.value;
              }
            }
          }
        } else if (item.type === 'B') {
          if ('values' in item && item.values) {
            for (const v of item.values) {
              if (v !== null && v !== undefined) {
                count += v;
              }
            }
          }
        }
      }
    }

    return count;
  }

  // Anti-pattern: Multiple returns in a method
  methodWithManyReturns(key: string): unknown {
    if (!key) {
      return null;
    }

    if (!(key in this.config)) {
      return 'not found';
    }

    const value = this.config[key];

    if (value === null) {
      return 'null';
    }

    if (typeof value === 'string') {
      return value.toUpperCase();
    }

    if (typeof value === 'number') {
      return value * 2;
    }

    return value;
  }
}

// Arrow function with too many params
const arrowWithTooManyParams = (
  a: number,
  b: number,
  c: number,
  d: number,
  e: number,
  f: number
): number => {
  return a + b + c + d + e + f;
};

// Function combining multiple anti-patterns
function combinedAntipatterns(
  a: number,
  b: number,
  c: number,
  d: number,
  e: number,
  f: number
): number {
  let result = 0;

  // Deep nesting
  if (a > 0) {
    if (b > 0) {
      if (c > 0) {
        if (d > 0) {
          if (e > 0) {
            // Too deep
            result = a + b + c + d + e + f;
          }
        }
      }
    }
  }

  // Magic numbers
  if (result < 42) {
    result = 100;
  } else if (result < 256) {
    result = 500;
  }

  // Multiple returns
  if (result === 0) {
    return -1;
  }

  if (result < 0) {
    return 0;
  }

  return result;
}

export {
  functionWithTooManyParams,
  deeplyNestedCode,
  tooManyReturns,
  magicNumbersEverywhere,
  calculatePrice,
  AntiPatternClass,
  arrowWithTooManyParams,
  combinedAntipatterns,
};
